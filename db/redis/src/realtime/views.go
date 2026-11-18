package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// How many times each blog post has been read, for the count shown under
// it on jungho.dev/blog. A view is one visitor reading one post once per
// window (a day in production); the proxy decides who a visitor is, from
// the same IP and User-Agent hash it uses for presence, and only asks about
// posts the blog lists. This store does not know which posts exist.
//
//	views:count                 Hash, field = slug, value = running total
//	views:seen:<slug>:<id>      String, expires with the window: "this
//	                            visitor was already counted for this post"
const (
	ViewsCountKey   = "views:count"
	ViewsSeenPrefix = "views:seen:"
	// MaxViewSlugs caps one VIEWS line. A blog page asks about the posts it
	// lists, which is far fewer.
	MaxViewSlugs = 100
)

var (
	slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	// The proxy sends sixteen hex characters. Anything short and plain is
	// accepted so the key stays readable; nothing long or with separators
	// gets into a key name.
	viewerPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
)

// ValidSlug reports whether s is shaped like a blog post's slug.
func ValidSlug(s string) bool {
	return slugPattern.MatchString(s)
}

// ViewRequest is the argument of a VIEW command.
type ViewRequest struct {
	Slug string `json:"slug"`
	ID   string `json:"id"`
}

// ParseView reads `{"slug":"...","id":"..."}`.
func ParseView(arg string) (ViewRequest, error) {
	if arg == "" {
		return ViewRequest{}, errors.New("VIEW needs a slug and a visitor id")
	}
	var v ViewRequest
	if err := json.Unmarshal([]byte(arg), &v); err != nil {
		return ViewRequest{}, fmt.Errorf("VIEW payload is not valid JSON: %w", err)
	}
	if !ValidSlug(v.Slug) {
		return ViewRequest{}, fmt.Errorf("VIEW slug %q is not a post slug", v.Slug)
	}
	if !viewerPattern.MatchString(v.ID) {
		return ViewRequest{}, fmt.Errorf("VIEW id %q is not a visitor id", v.ID)
	}
	return v, nil
}

// ParseSlugs reads the comma-separated argument of a VIEWS command, keeping
// the order and dropping repeats.
func ParseSlugs(arg string) ([]string, error) {
	if strings.TrimSpace(arg) == "" {
		return nil, errors.New("VIEWS needs at least one slug")
	}
	seen := map[string]bool{}
	var slugs []string
	for _, part := range strings.Split(arg, ",") {
		slug := strings.TrimSpace(part)
		if !ValidSlug(slug) {
			return nil, fmt.Errorf("VIEWS slug %q is not a post slug", slug)
		}
		if seen[slug] {
			continue
		}
		seen[slug] = true
		slugs = append(slugs, slug)
	}
	if len(slugs) > MaxViewSlugs {
		return nil, fmt.Errorf("VIEWS takes at most %d slugs, got %d", MaxViewSlugs, len(slugs))
	}
	return slugs, nil
}

// ViewStore keeps the view counts in Redis.
type ViewStore struct {
	client *redis.Client
	window time.Duration
}

func NewViewStore(client *redis.Client, window time.Duration) *ViewStore {
	return &ViewStore{client: client, window: window}
}

// viewScript marks the visitor as counted for the window and adds the view,
// or, when the mark is already there, reads the total. KEYS: the marker,
// views:count. ARGV: the slug, the window in milliseconds. Redis runs a
// script as one step, so a view is never half done: before this it was SET
// NX and then HINCRBY, and a failure between the two left the marker without
// the count, so that reader went uncounted for a day.
const viewScript = `
if redis.call('SET', KEYS[1], '1', 'NX', 'PX', ARGV[2]) then
  return redis.call('HINCRBY', KEYS[2], ARGV[1], 1)
end
return tonumber(redis.call('HGET', KEYS[2], ARGV[1]) or '0')
`

// View counts one read of slug by visitor id, unless that visitor was
// already counted for it inside the window, and returns the total either
// way. SET NX is what makes two tabs opened at once count once: only one of
// them creates the marker. EVAL rather than EVALSHA so it is always one
// round trip; the script is a few hundred bytes.
func (s *ViewStore) View(ctx context.Context, slug, id string) (int64, error) {
	marker := ViewsSeenPrefix + slug + ":" + id
	return s.client.Eval(ctx, viewScript,
		[]string{marker, ViewsCountKey},
		slug, s.window.Milliseconds(),
	).Int64()
}

// Counts returns the total for each slug; a slug nobody has read is 0.
func (s *ViewStore) Counts(ctx context.Context, slugs []string) (map[string]int64, error) {
	values, err := s.client.HMGet(ctx, ViewsCountKey, slugs...).Result()
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int64, len(slugs))
	for i, slug := range slugs {
		counts[slug] = 0
		raw, ok := values[i].(string)
		if !ok {
			continue
		}
		var n int64
		if _, err := fmt.Sscan(raw, &n); err != nil {
			return nil, fmt.Errorf("views:count %s holds %q: %w", slug, raw, err)
		}
		counts[slug] = n
	}
	return counts, nil
}
