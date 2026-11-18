package realtime

import (
	"fmt"
	"strings"
	"testing"
)

func TestValidSlug(t *testing.T) {
	for _, s := range []string{"rfq-deep-dive", "a", "v2"} {
		if !ValidSlug(s) {
			t.Errorf("%q should be valid", s)
		}
	}
	for _, s := range []string{"", "RFQ", "a--b", "-a", "a-", "a/b", "../x", "a b"} {
		if ValidSlug(s) {
			t.Errorf("%q should be invalid", s)
		}
	}
}

func TestParseView(t *testing.T) {
	got, err := ParseView(`{"slug":"rfq-deep-dive","id":"0123456789abcdef"}`)
	if err != nil {
		t.Fatal(err)
	}
	if got != (ViewRequest{Slug: "rfq-deep-dive", ID: "0123456789abcdef"}) {
		t.Errorf("got %+v", got)
	}

	for _, arg := range []string{
		``,
		`rfq-deep-dive`,
		`{"slug":"rfq-deep-dive"}`,
		`{"slug":"RFQ","id":"x"}`,
		`{"slug":"a","id":"has space"}`,
		`{"slug":"a","id":"` + strings.Repeat("x", 65) + `"}`,
	} {
		if _, err := ParseView(arg); err == nil {
			t.Errorf("ParseView(%q) should fail", arg)
		}
	}
}

func TestParseSlugs(t *testing.T) {
	cases := map[string][]string{
		"a,b":           {"a", "b"},
		" a , b ":       {"a", "b"},
		"a,b,a":         {"a", "b"},
		"rfq-deep-dive": {"rfq-deep-dive"},
	}
	for arg, want := range cases {
		got, err := ParseSlugs(arg)
		if err != nil {
			t.Errorf("ParseSlugs(%q): %v", arg, err)
			continue
		}
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("ParseSlugs(%q) = %v, want %v", arg, got, want)
		}
	}

	many := make([]string, MaxViewSlugs+1)
	for i := range many {
		many[i] = fmt.Sprintf("post-%d", i)
	}
	for _, arg := range []string{"", "a,,b", "a,RFQ", strings.Join(many, ",")} {
		if _, err := ParseSlugs(arg); err == nil {
			t.Errorf("ParseSlugs(%.40q) should fail", arg)
		}
	}
}
