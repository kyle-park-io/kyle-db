package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// A session is one visitor: whoever is holding a websocket open. The proxy
// decides what "one visitor" means and sends us the id; see
// ingress-reverse-proxy/redis/identity.go.
//
// Two keys back it:
//
//	online:sessions      ZSet, member = session id, score = unix expiry
//	online:meta:<id>     Hash, that session's country/browser/os/path
//
// The ZSet answers "who is here" in one command and expires members by
// score. The hashes carry what we show about them, and each carries its own
// Redis TTL so a hash cannot outlive its ZSet member even if a cleanup is
// missed.
const (
	SessionsKey = "online:sessions"
	metaPrefix  = "online:meta:"
)

// Session is what a client reports about itself.
type Session struct {
	ID      string `json:"id"`
	Country string `json:"country,omitempty"`
	Browser string `json:"browser,omitempty"`
	OS      string `json:"os,omitempty"`
	Path    string `json:"path,omitempty"`
}

// Bucket is one row of a breakdown: a label and how many sessions carry it.
type Bucket struct {
	Key string `json:"key"`
	N   int    `json:"n"`
}

// Snapshot is the whole picture, and is what REAL-TIME returns.
type Snapshot struct {
	Count     int      `json:"count"`
	Countries []Bucket `json:"countries"`
	Pages     []Bucket `json:"pages"`
}

// SessionStore keeps the live sessions in Redis.
type SessionStore struct {
	client *redis.Client
	ttl    time.Duration
}

func NewSessionStore(client *redis.Client, ttl time.Duration) *SessionStore {
	return &SessionStore{client: client, ttl: ttl}
}

// ParseSession reads the argument of an ADDUSER command.
//
// The argument is JSON. Anything that is not JSON is taken as a bare session
// id with no metadata, which is what the previous protocol sent - the proxy
// and this server deploy as separate images, so for the length of a rollout
// one of them is always the old one.
func ParseSession(arg string) (Session, error) {
	if arg == "" {
		return Session{}, fmt.Errorf("ADDUSER needs a session")
	}
	if arg[0] != '{' {
		return Session{ID: arg}, nil
	}
	var s Session
	if err := json.Unmarshal([]byte(arg), &s); err != nil {
		return Session{}, fmt.Errorf("ADDUSER payload is not valid JSON: %w", err)
	}
	if s.ID == "" {
		return Session{}, fmt.Errorf("ADDUSER payload has no id")
	}
	return s, nil
}

// Touch records a session as present, or refreshes one already present.
// Callers refresh well inside the TTL; see the proxy's heartbeat.
func (s *SessionStore) Touch(ctx context.Context, sess Session) error {
	expiry := time.Now().Add(s.ttl)

	pipe := s.client.TxPipeline()
	pipe.ZAdd(ctx, SessionsKey, redis.Z{
		Score:  float64(expiry.Unix()),
		Member: sess.ID,
	})

	meta := metaPrefix + sess.ID
	pipe.HSet(ctx, meta,
		"country", sess.Country,
		"browser", sess.Browser,
		"os", sess.OS,
		"path", sess.Path,
	)
	// HSetNX so a reconnecting session keeps the time it first arrived.
	pipe.HSetNX(ctx, meta, "first_seen", strconv.FormatInt(time.Now().Unix(), 10))
	// A little longer than the ZSet score, so the hash is still readable for
	// a snapshot taken in the same second the member expires.
	pipe.Expire(ctx, meta, s.ttl+10*time.Second)

	_, err := pipe.Exec(ctx)
	return err
}

// Drop removes a session immediately, without waiting for its score to pass.
// The proxy calls this the moment a websocket closes, so someone who leaves
// stops being counted at once instead of lingering for a whole TTL.
func (s *SessionStore) Drop(ctx context.Context, id string) error {
	pipe := s.client.TxPipeline()
	pipe.ZRem(ctx, SessionsKey, id)
	pipe.Del(ctx, metaPrefix+id)
	_, err := pipe.Exec(ctx)
	return err
}

// Snapshot removes whatever has expired and returns the current picture.
func (s *SessionStore) Snapshot(ctx context.Context) (Snapshot, error) {
	now := time.Now().Unix()

	// Everything scored at or before now is gone. Sweeping before reading
	// means the count and the breakdowns describe the same instant.
	if err := s.client.ZRemRangeByScore(
		ctx, SessionsKey, "-inf", fmt.Sprintf("(%d", now),
	).Err(); err != nil {
		return Snapshot{}, fmt.Errorf("sweeping expired sessions: %w", err)
	}

	ids, err := s.client.ZRangeByScore(ctx, SessionsKey, &redis.ZRangeBy{
		Min: strconv.FormatInt(now, 10),
		Max: "+inf",
	}).Result()
	if err != nil {
		return Snapshot{}, fmt.Errorf("listing sessions: %w", err)
	}
	if len(ids) == 0 {
		return Aggregate(nil), nil
	}

	// One round trip for every session's metadata rather than one per
	// session: this runs on a timer, and the timer used to be the busiest
	// thing talking to Redis.
	pipe := s.client.Pipeline()
	cmds := make([]*redis.MapStringStringCmd, len(ids))
	for i, id := range ids {
		cmds[i] = pipe.HGetAll(ctx, metaPrefix+id)
	}
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return Snapshot{}, fmt.Errorf("reading session metadata: %w", err)
	}

	sessions := make([]Session, 0, len(ids))
	for i, id := range ids {
		m, err := cmds[i].Result()
		if err != nil && err != redis.Nil {
			// One unreadable hash is not worth losing the whole snapshot;
			// the session still counts, just without its breakdown.
			m = nil
		}
		sessions = append(sessions, Session{
			ID:      id,
			Country: m["country"],
			Browser: m["browser"],
			OS:      m["os"],
			Path:    m["path"],
		})
	}
	return Aggregate(sessions), nil
}

// Aggregate turns a list of sessions into the snapshot the clients render.
// Pure, so the shape of the answer is testable without a Redis.
//
// Buckets are ordered by size and then by key, so a snapshot with the same
// sessions is byte-identical every time - clients diff these to decide
// whether anything changed, and Redis returns no order worth trusting.
func Aggregate(sessions []Session) Snapshot {
	countries := map[string]int{}
	pages := map[string]int{}
	for _, s := range sessions {
		if s.Country != "" {
			countries[s.Country]++
		}
		if s.Path != "" {
			pages[s.Path]++
		}
	}
	return Snapshot{
		Count:     len(sessions),
		Countries: buckets(countries),
		Pages:     buckets(pages),
	}
}

func buckets(counts map[string]int) []Bucket {
	out := make([]Bucket, 0, len(counts))
	for k, n := range counts {
		out = append(out, Bucket{Key: k, N: n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].N != out[j].N {
			return out[i].N > out[j].N
		}
		return out[i].Key < out[j].Key
	})
	return out
}
