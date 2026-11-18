package realtime

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// views connects to the Redis named by REDIS_ADDR on its own database: the
// session tests use 14 and the protocol tests 15, and each flushes before it
// starts.
func views(t *testing.T, window time.Duration) (*ViewStore, *redis.Client, context.Context) {
	t.Helper()
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		t.Skip("set REDIS_ADDR to run the Redis-backed tests")
	}
	client := redis.NewClient(&redis.Options{Addr: addr, DB: 13})
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("no Redis at %s: %v", addr, err)
	}
	if err := client.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("flushing: %v", err)
	}
	t.Cleanup(func() { client.Close() })
	return NewViewStore(client, window), client, ctx
}

func view(t *testing.T, s *ViewStore, ctx context.Context, slug, id string) int64 {
	t.Helper()
	n, err := s.View(ctx, slug, id)
	if err != nil {
		t.Fatalf("View(%s, %s): %v", slug, id, err)
	}
	return n
}

func TestAVisitorCountsOncePerWindow(t *testing.T) {
	s, _, ctx := views(t, 24*time.Hour)
	for i := 0; i < 3; i++ {
		if got := view(t, s, ctx, "a", "v1"); got != 1 {
			t.Fatalf("view %d answered %d, want 1", i, got)
		}
	}
}

func TestASecondVisitorCounts(t *testing.T) {
	s, _, ctx := views(t, 24*time.Hour)
	view(t, s, ctx, "a", "v1")
	if got := view(t, s, ctx, "a", "v2"); got != 2 {
		t.Errorf("second visitor answered %d, want 2", got)
	}
}

func TestTwoTabsAtOnceCountOnce(t *testing.T) {
	s, client, ctx := views(t, 24*time.Hour)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.View(ctx, "a", "v1"); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	got, err := client.HGet(ctx, ViewsCountKey, "a").Int64()
	if err != nil {
		t.Fatal(err)
	}
	if got != 1 {
		t.Errorf("20 simultaneous views from one visitor counted %d, want 1", got)
	}
}

func TestAnExpiredMarkerCountsAgain(t *testing.T) {
	s, client, ctx := views(t, 300*time.Millisecond)
	view(t, s, ctx, "a", "v1")
	// Wait for Redis to expire the marker rather than for a fixed 400ms:
	// Redis expires by the wall clock, and WSL moved it back 2.8s twice in
	// one minute on the desktop (2026-11-19), so a fixed sleep sometimes
	// woke before the marker was gone.
	waitUntil(t, 8*time.Second, func() bool {
		n, err := client.Exists(ctx, ViewsSeenPrefix+"a:v1").Result()
		return err == nil && n == 0
	})
	if got := view(t, s, ctx, "a", "v1"); got != 2 {
		t.Errorf("view after the window answered %d, want 2", got)
	}
}

// waitUntil polls ok every 50ms and fails the test if it is still false
// after limit.
func waitUntil(t *testing.T, limit time.Duration, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(limit)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatalf("still not true after %v", limit)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// roundTrips counts commands and pipelines sent to Redis.
type roundTrips struct{ n atomic.Int64 }

func (h *roundTrips) DialHook(next redis.DialHook) redis.DialHook { return next }

func (h *roundTrips) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		h.n.Add(1)
		return next(ctx, cmd)
	}
}

func (h *roundTrips) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		h.n.Add(1)
		return next(ctx, cmds)
	}
}

// A view used to be SET NX and then HINCRBY (or HGET): two round trips, and
// a failure between them left the marker without the count, so that reader
// went uncounted for a day. One script makes it one atomic step.
func TestAViewIsOneRoundTrip(t *testing.T) {
	s, client, ctx := views(t, 24*time.Hour)
	h := &roundTrips{}
	client.AddHook(h)

	if got := view(t, s, ctx, "a", "v1"); got != 1 {
		t.Fatalf("first view answered %d, want 1", got)
	}
	if got := view(t, s, ctx, "a", "v1"); got != 1 {
		t.Fatalf("repeat view answered %d, want 1", got)
	}
	if got := h.n.Load(); got != 2 {
		t.Errorf("two views took %d round trips to Redis, want 2", got)
	}
}

func TestTheMarkerExpiresWithTheWindow(t *testing.T) {
	s, client, ctx := views(t, 24*time.Hour)
	view(t, s, ctx, "a", "v1")
	ttl, err := client.PTTL(ctx, ViewsSeenPrefix+"a:v1").Result()
	if err != nil {
		t.Fatal(err)
	}
	if ttl <= 24*time.Hour-time.Minute || ttl > 24*time.Hour {
		t.Errorf("marker TTL = %v, want just under 24h", ttl)
	}
}

func TestCountsReturnsZeroForUnknownSlugs(t *testing.T) {
	s, _, ctx := views(t, 24*time.Hour)
	view(t, s, ctx, "a", "v1")
	got, err := s.Counts(ctx, []string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	if got["a"] != 1 || got["b"] != 0 || len(got) != 2 {
		t.Errorf("Counts = %v, want map[a:1 b:0]", got)
	}
}
