package realtime

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestParseSessionReadsJSON(t *testing.T) {
	got, err := ParseSession(`{"id":"abc123","country":"KR","browser":"Chrome","os":"macOS","path":"/blog"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := Session{ID: "abc123", Country: "KR", Browser: "Chrome", OS: "macOS", Path: "/blog"}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestParseSessionAcceptsTheOldBareIDForm(t *testing.T) {
	// The proxy and this server are separate images, so during any rollout
	// one of them is the old one. A bare argument is the old ADDUSER.
	got, err := ParseSession("104.23.251.104:13965")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "104.23.251.104:13965" {
		t.Errorf("got id %q", got.ID)
	}
	if got.Country != "" || got.Path != "" {
		t.Errorf("a bare id carries no metadata, got %+v", got)
	}
}

func TestParseSessionRejectsEmptyAndIDLess(t *testing.T) {
	for name, arg := range map[string]string{
		"empty":         "",
		"json   no id":  `{"country":"KR"}`,
		"broken   json": `{"id":`,
	} {
		if _, err := ParseSession(arg); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestAggregateCountsSessionsNotRequests(t *testing.T) {
	// The bug this whole change exists for: one visitor loading one page
	// used to arrive as ~13 members, because the member was the Cloudflare
	// edge address:port of each individual request. One visitor is one
	// session id now, however many requests they make.
	got := Aggregate([]Session{
		{ID: "a", Country: "KR", Path: "/blog"},
		{ID: "b", Country: "KR", Path: "/blog"},
		{ID: "c", Country: "US", Path: "/"},
	})
	if got.Count != 3 {
		t.Errorf("count = %d, want 3", got.Count)
	}
	wantCountries := []Bucket{{Key: "KR", N: 2}, {Key: "US", N: 1}}
	if !reflect.DeepEqual(got.Countries, wantCountries) {
		t.Errorf("countries = %+v, want %+v", got.Countries, wantCountries)
	}
	wantPages := []Bucket{{Key: "/blog", N: 2}, {Key: "/", N: 1}}
	if !reflect.DeepEqual(got.Pages, wantPages) {
		t.Errorf("pages = %+v, want %+v", got.Pages, wantPages)
	}
}

func TestAggregateOrdersBySizeThenKey(t *testing.T) {
	// Clients diff these snapshots to decide whether to re-render, and Redis
	// returns no order worth trusting, so equal-sized buckets must still
	// come back in one fixed order.
	got := Aggregate([]Session{
		{ID: "1", Country: "US"},
		{ID: "2", Country: "KR"},
		{ID: "3", Country: "JP"},
		{ID: "4", Country: "KR"},
	})
	want := []Bucket{{Key: "KR", N: 2}, {Key: "JP", N: 1}, {Key: "US", N: 1}}
	if !reflect.DeepEqual(got.Countries, want) {
		t.Errorf("got %+v, want %+v", got.Countries, want)
	}
}

func TestAggregateSkipsUnknownFields(t *testing.T) {
	// A session that arrived over the old protocol has no country and no
	// path. It still counts as a person; it just adds no bucket.
	got := Aggregate([]Session{
		{ID: "known", Country: "KR", Path: "/blog"},
		{ID: "legacy"},
	})
	if got.Count != 2 {
		t.Errorf("count = %d, want 2", got.Count)
	}
	if len(got.Countries) != 1 || got.Countries[0].N != 1 {
		t.Errorf("countries = %+v, want one bucket of 1", got.Countries)
	}
}

func TestEmptyAggregateMarshalsAsEmptyArraysNotNull(t *testing.T) {
	// The clients iterate these. `null` would make them throw; `[]` renders
	// as nothing, which is what an empty site should look like.
	blob, err := json.Marshal(Aggregate(nil))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"count":0,"countries":[],"pages":[]}`
	if string(blob) != want {
		t.Errorf("got %s, want %s", blob, want)
	}
}
