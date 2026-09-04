package realtime

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// store connects to the Redis named by REDIS_ADDR, and skips when there is
// none. The pure logic is covered in session_test.go; this file is for the
// parts that only exist in terms of Redis - what the ZSet and the hashes do
// together, and whether an expiry actually removes somebody.
func store(t *testing.T, ttl time.Duration) (*SessionStore, context.Context) {
	t.Helper()
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		t.Skip("set REDIS_ADDR to run the Redis-backed tests")
	}

	// Its own database. `go test ./...` runs packages in parallel, and this
	// file and handler/protocol_test.go both flush before they start - on a
	// shared database each was deleting the other's sessions mid-test.
	client := redis.NewClient(&redis.Options{Addr: addr, DB: 14})
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("no Redis at %s: %v", addr, err)
	}
	if err := client.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("flushing: %v", err)
	}
	t.Cleanup(func() { client.Close() })

	return NewSessionStore(client, ttl), ctx
}

func TestOneVisitorStaysOneVisitorHoweverOftenTheyRefresh(t *testing.T) {
	// This is the reported bug, at the level it was caused: the count used
	// to rise with every request a visitor made.
	s, ctx := store(t, 30*time.Second)

	for i := 0; i < 20; i++ {
		if err := s.Touch(ctx, Session{ID: "visitor-a", Country: "KR", Path: "/blog"}); err != nil {
			t.Fatalf("touch %d: %v", i, err)
		}
	}

	got, err := s.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.Count != 1 {
		t.Errorf("count = %d after 20 refreshes from one visitor, want 1", got.Count)
	}
}

func TestSnapshotReportsWhereAndWhatEveryoneIsReading(t *testing.T) {
	s, ctx := store(t, 30*time.Second)

	for _, sess := range []Session{
		{ID: "a", Country: "KR", Browser: "Chrome", OS: "macOS", Path: "/blog"},
		{ID: "b", Country: "KR", Browser: "Safari", OS: "iOS", Path: "/blog"},
		{ID: "c", Country: "US", Browser: "Firefox", OS: "Linux", Path: "/"},
	} {
		if err := s.Touch(ctx, sess); err != nil {
			t.Fatal(err)
		}
	}

	got, err := s.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.Count != 3 {
		t.Fatalf("count = %d, want 3", got.Count)
	}
	if len(got.Countries) != 2 || got.Countries[0].Key != "KR" || got.Countries[0].N != 2 {
		t.Errorf("countries = %+v", got.Countries)
	}
	if len(got.Pages) != 2 || got.Pages[0].Key != "/blog" || got.Pages[0].N != 2 {
		t.Errorf("pages = %+v", got.Pages)
	}
}

func TestASessionExpiresWhenItStopsRefreshing(t *testing.T) {
	// A one-second TTL stands in for the real thirty.
	s, ctx := store(t, 1*time.Second)

	if err := s.Touch(ctx, Session{ID: "leaving", Country: "KR"}); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Snapshot(ctx); got.Count != 1 {
		t.Fatalf("count = %d immediately after touch, want 1", got.Count)
	}

	time.Sleep(2100 * time.Millisecond)

	got, err := s.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.Count != 0 {
		t.Errorf("count = %d after the TTL passed, want 0", got.Count)
	}
	if len(got.Countries) != 0 {
		t.Errorf("an expired session left a breakdown behind: %+v", got.Countries)
	}
}

func TestARefreshWithinTheTTLKeepsAVisitorCounted(t *testing.T) {
	// The second half of the reported bug: the refresh beat and the TTL were
	// both 30 seconds, so a visitor who never left still blinked out of the
	// count. A refresh at half the TTL must hold them.
	s, ctx := store(t, 2*time.Second)

	if err := s.Touch(ctx, Session{ID: "staying", Country: "KR"}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		time.Sleep(1 * time.Second)
		if err := s.Touch(ctx, Session{ID: "staying", Country: "KR"}); err != nil {
			t.Fatal(err)
		}
		got, err := s.Snapshot(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if got.Count != 1 {
			t.Fatalf("count = %d at refresh %d, want 1 throughout", got.Count, i)
		}
	}
}

func TestDropRemovesAVisitorAtOnce(t *testing.T) {
	// A closed tab should stop being counted immediately, not after a TTL.
	s, ctx := store(t, 30*time.Second)

	if err := s.Touch(ctx, Session{ID: "gone", Country: "KR", Path: "/blog"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Drop(ctx, "gone"); err != nil {
		t.Fatal(err)
	}

	got, err := s.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.Count != 0 {
		t.Errorf("count = %d after dropping the only visitor, want 0", got.Count)
	}
	if len(got.Countries) != 0 || len(got.Pages) != 0 {
		t.Errorf("drop left metadata behind: %+v", got)
	}
}

func TestFirstSeenSurvivesARefresh(t *testing.T) {
	// The hover card shows how long someone has been here, so a refresh must
	// not reset the clock.
	s, ctx := store(t, 30*time.Second)

	if err := s.Touch(ctx, Session{ID: "steady", Country: "KR"}); err != nil {
		t.Fatal(err)
	}
	first, err := s.client.HGet(ctx, "online:meta:steady", "first_seen").Result()
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(1100 * time.Millisecond)
	if err := s.Touch(ctx, Session{ID: "steady", Country: "KR"}); err != nil {
		t.Fatal(err)
	}

	again, err := s.client.HGet(ctx, "online:meta:steady", "first_seen").Result()
	if err != nil {
		t.Fatal(err)
	}
	if first != again {
		t.Errorf("first_seen moved from %s to %s across a refresh", first, again)
	}
}
