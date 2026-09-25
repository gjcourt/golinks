package domain

import (
	"reflect"
	"testing"
)

func TestValidWildcardShortcode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, input string
		want        bool
	}{
		// Legacy single trailing "*".
		{"simple", "pulls/*", true},
		{"jira", "jira/*", true},
		{"multi-literal", "gh/pr/*", true},
		{"hyphen-underscore-segments", "my-repo/pull_requests/*", true},
		{"bare-star-rejected", "*", false},
		{"star-not-final", "pulls/*/x", false},
		{"two-stars", "pu*ls/*", false},
		{"two-star-segments", "gh/*/*", false},
		{"no-star", "pulls/x", false},
		{"trailing-slash-no-star", "pulls/", false},
		{"star-mid-segment", "pull*/*", false},
		{"empty-literal-segment", "pulls//*", false},
		{"bad-char-in-literal", "pu lls/*", false},

		// Named parameters.
		{"named-two", "gh/{repo}/{pr}", true},
		{"named-one", "jira/{ticket}", true},
		{"named-literal-between", "gh/{repo}/pull/{n}", true},
		{"named-underscore-digit", "x/{a_1}", true},
		{"named-first-segment", "{repo}/x", false},
		{"named-repeated", "gh/{a}/{a}", false},
		{"named-mixed-with-star", "gh/{a}/*", false},
		{"star-then-named", "gh/*/{a}", false},
		{"named-partial-segment", "gh/x{a}", false},
		{"named-uppercase", "gh/{Repo}", false},
		{"named-leading-digit", "gh/{1a}", false},
		{"named-empty", "gh/{}", false},
		{"named-unclosed", "gh/{repo", false},
		{"named-hyphen", "gh/{my-repo}", false},
		{"named-too-long-name", "gh/{abcdefghijklmnopqrstuvwxyzabcdefg}", false},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ValidWildcardShortcode(tt.input); got != tt.want {
				t.Errorf("ValidWildcardShortcode(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestMatchWildcard(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, shortcode, path string
		want                  map[string]string
		wantOK                bool
	}{
		{"basic", "pulls/*", "pulls/10", map[string]string{"*": "10"}, true},
		{"dotted-capture", "release/*", "release/v1.2.3", map[string]string{"*": "v1.2.3"}, true},
		{"hyphen-capture", "u/*", "u/john-doe", map[string]string{"*": "john-doe"}, true},
		{"multi-segment-no-match", "pulls/*", "pulls/10/extra", nil, false},
		{"trailing-slash-no-match", "pulls/*", "pulls/10/", nil, false},
		{"empty-capture-no-match", "pulls/*", "pulls/", nil, false},
		{"prefix-mismatch", "pulls/*", "issues/10", nil, false},
		{"dotdot-rejected", "pulls/*", "pulls/..", nil, false},
		{"dot-rejected", "pulls/*", "pulls/.", nil, false},
		{"encoded-percent-rejected", "pulls/*", "pulls/%2e%2e", nil, false},
		{"space-rejected", "pulls/*", "pulls/a b", nil, false},
		{"colon-rejected", "pulls/*", "pulls/http:", nil, false},

		{"named-two", "gh/{repo}/{pr}", "gh/api/10", map[string]string{"repo": "api", "pr": "10"}, true},
		{"named-literal-between", "gh/{repo}/pull/{n}", "gh/api/pull/7", map[string]string{"repo": "api", "n": "7"}, true},
		{"named-literal-mismatch", "gh/{repo}/pull/{n}", "gh/api/issue/7", nil, false},
		{"named-too-few", "gh/{repo}/{pr}", "gh/api", nil, false},
		{"named-too-many", "gh/{repo}/{pr}", "gh/api/10/x", nil, false},
		{"named-unsafe-capture", "gh/{repo}/{pr}", "gh/..%2f/10", nil, false},
		{"named-literal-text", "gh/{repo}/{pr}", "gh/{repo}/{pr}", nil, false},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := MatchWildcard(tt.shortcode, tt.path)
			if ok != tt.wantOK || (ok && !reflect.DeepEqual(got, tt.want)) {
				t.Errorf("MatchWildcard(%q, %q) = (%v, %v), want (%v, %v)",
					tt.shortcode, tt.path, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestResolveWildcard_LongestPrefixWins(t *testing.T) {
	t.Parallel()
	links := []*Link{
		{Shortcode: "gh/*", URL: "https://example.com/a/*"},
		{Shortcode: "gh/pr/*", URL: "https://example.com/b/*"},
	}
	// "gh/pr/123" is matched only by "gh/pr/*" (the "gh/*" pattern requires a
	// single segment after "gh/", so it does not match a two-segment tail).
	match, caps, ok := ResolveWildcard(links, "gh/pr/123")
	if !ok || match.Shortcode != "gh/pr/*" || caps["*"] != "123" {
		t.Fatalf("got (%v, %v, %v), want (gh/pr/*, 123)", match, caps, ok)
	}
	// A single-segment tail matches only the shorter pattern.
	match, caps, ok = ResolveWildcard(links, "gh/xyz")
	if !ok || match.Shortcode != "gh/*" || caps["*"] != "xyz" {
		t.Errorf("got (%v, %v, %v), want (gh/*, xyz, true)", match, caps, ok)
	}
}

// A path both a named and a more literal pattern match goes to the one with
// more literal segments, whatever the storage order.
func TestResolveWildcard_MoreLiteralsWins(t *testing.T) {
	t.Parallel()
	generic := &Link{Shortcode: "gh/{repo}/{pr}", URL: "https://example.com/{repo}/pull/{pr}"}
	specific := &Link{Shortcode: "gh/homelab/{pr}", URL: "https://example.com/homelab/pull/{pr}"}
	for _, order := range [][]*Link{{generic, specific}, {specific, generic}} {
		m, caps, ok := ResolveWildcard(order, "gh/homelab/5")
		if !ok || m != specific || caps["pr"] != "5" {
			t.Errorf("gh/homelab/5 -> (%v, %v, %v), want the gh/homelab/{pr} link", m, caps, ok)
		}
		m, caps, ok = ResolveWildcard(order, "gh/api/5")
		if !ok || m != generic || caps["repo"] != "api" || caps["pr"] != "5" {
			t.Errorf("gh/api/5 -> (%v, %v, %v), want the generic link", m, caps, ok)
		}
	}
}

func TestResolveWildcard_OrderIndependent(t *testing.T) {
	t.Parallel()
	short := &Link{Shortcode: "gh/*", URL: "https://example.com/a/*"}
	long := &Link{Shortcode: "gh/pr/*", URL: "https://example.com/b/*"}
	for _, order := range [][]*Link{{short, long}, {long, short}} {
		m, caps, ok := ResolveWildcard(order, "gh/pr/9")
		if !ok || m.Shortcode != "gh/pr/*" || caps["*"] != "9" {
			t.Errorf("order %v -> (%v, %v, %v), want (gh/pr/*, 9, true)", order, m, caps, ok)
		}
	}
	// Equal specificity: the lexicographically smallest shortcode wins.
	a := &Link{Shortcode: "x/{a}", URL: "https://example.com/{a}"}
	b := &Link{Shortcode: "x/{b}", URL: "https://example.com/{b}"}
	for _, order := range [][]*Link{{a, b}, {b, a}} {
		if m, _, ok := ResolveWildcard(order, "x/1"); !ok || m != a {
			t.Errorf("tie -> %v, want x/{a}", m)
		}
	}
}

func TestResolveWildcard_NoMatch(t *testing.T) {
	t.Parallel()
	links := []*Link{
		{Shortcode: "pulls/*", URL: "https://example.com/pull/*"},
		{Shortcode: "gh/{repo}/{pr}", URL: "https://example.com/{repo}/{pr}"},
		{Shortcode: "docs", URL: "https://example.com/docs"}, // non-pattern, ignored
	}
	if m, _, ok := ResolveWildcard(links, "issues/1"); ok {
		t.Errorf("issues/1 unexpectedly matched %v", m)
	}
	// A pattern visited by its own literal text does not resolve (the capture
	// fails the safe-charset rule).
	for _, p := range []string{"pulls/*", "gh/{repo}/{pr}"} {
		if m, _, ok := ResolveWildcard(links, p); ok {
			t.Errorf("literal pattern path %q matched %v", p, m)
		}
	}
}

func TestSubstituteWildcard(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, template string
		caps           map[string]string
		want           string
		wantErr        bool
	}{
		{"basic", "https://github.com/gjcourt/homelab/pull/*", map[string]string{"*": "10"}, "https://github.com/gjcourt/homelab/pull/10", false},
		{"dotted", "https://example.com/r/*", map[string]string{"*": "v1.2.3"}, "https://example.com/r/v1.2.3", false},
		{"named", "https://github.com/acme/{repo}/pull/{pr}", map[string]string{"repo": "api", "pr": "10"}, "https://github.com/acme/api/pull/10", false},
		{"named-repeated", "https://example.com/{a}?q={a}", map[string]string{"a": "x"}, "https://example.com/x?q=x", false},
		{"unsafe-slash", "https://example.com/*", map[string]string{"*": "a/b"}, "", true},
		{"unsafe-percent", "https://example.com/*", map[string]string{"*": "%2e%2e"}, "", true},
		{"unsafe-colon", "https://example.com/{a}", map[string]string{"a": "http:"}, "", true},
		{"empty-capture", "https://example.com/*", map[string]string{"*": ""}, "", true},
		{"missing-capture", "https://example.com/{a}/{b}", map[string]string{"a": "x"}, "", true},
		// A legacy template leaves literal {…} alone (e.g. in a query string).
		{"legacy-literal-braces", "https://ex.com/search?t={type}&q=*", map[string]string{"*": "go"}, "https://ex.com/search?t={type}&q=go", false},
		// A template letting a capture reach the host is refused at redirect
		// time, however it was stored.
		{"host-from-capture-legacy", "https://*/", map[string]string{"*": "evil.com"}, "", true},
		{"host-from-capture-named", "https://{h}/x", map[string]string{"h": "evil.com"}, "", true},
		{"host-suffix-from-capture", "https://ex*/", map[string]string{"*": "ample.evil.com"}, "", true},
		{"userinfo-from-capture", "https://*@good.com/", map[string]string{"*": "evil.com"}, "", true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := SubstituteWildcard(tt.template, tt.caps)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("SubstituteWildcard(%q, %v) = %q, want error", tt.template, tt.caps, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("SubstituteWildcard(%q, %v) = %q, want %q", tt.template, tt.caps, got, tt.want)
			}
		})
	}
}

func TestNormalizeWildcardURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, shortcode, input, want string
		wantErr                      bool
	}{
		{"path-star", "pulls/*", "https://github.com/gjcourt/homelab/pull/*", "https://github.com/gjcourt/homelab/pull/*", false},
		{"no-scheme", "pulls/*", "example.com/pull/*", "https://example.com/pull/*", false},
		{"query-star", "s/*", "https://example.com/search?q=*", "https://example.com/search?q=*", false},
		{"named", "gh/{repo}/{pr}", "https://github.com/intrinsic-org/{repo}/pull/{pr}", "https://github.com/intrinsic-org/{repo}/pull/{pr}", false},
		{"named-reordered", "gh/{repo}/{pr}", "https://example.com/{pr}/{repo}", "https://example.com/{pr}/{repo}", false},
		{"named-repeated", "x/{a}", "https://example.com/{a}?also={a}", "https://example.com/{a}?also={a}", false},
		{"named-no-scheme", "x/{a}", "example.com/p/{a}", "https://example.com/p/{a}", false},

		// Rejections.
		{"no-star", "pulls/*", "https://example.com/pull/x", "", true},
		{"two-stars", "pulls/*", "https://example.com/*/*", "", true},
		{"star-in-host", "pulls/*", "https://*.example.com/x", "", true},
		{"bare-star", "pulls/*", "*", "", true},
		{"star-is-host", "pulls/*", "https://*", "", true},
		{"javascript-scheme", "pulls/*", "javascript:*", "", true},
		{"named-unused-param", "gh/{repo}/{pr}", "https://example.com/{repo}", "", true},
		{"named-unknown-placeholder", "x/{a}", "https://example.com/{a}/{b}", "", true},
		{"named-in-host", "x/{a}", "https://{a}.example.com/", "", true},
		{"named-is-host", "x/{a}", "https://{a}/", "", true},
		{"named-with-star", "x/{a}", "https://example.com/{a}/*", "", true},
		{"stray-brace", "x/{a}", "https://example.com/{a}/}", "", true},
		{"uppercase-placeholder", "x/{a}", "https://example.com/{A}", "", true},
		{"legacy-named-placeholder", "pulls/*", "https://example.com/{a}", "", true},
		{"sentinel-injection", "x/{a}", "https://example.com/golinksparamsentinelaaa/{a}", "", true},
		{"named-in-userinfo", "x/{a}", "https://{a}@evil.com/", "", true},
		{"named-in-port", "x/{a}", "https://example.com:{a}/", "", true},
		{"named-http-host", "x/{a}", "http://{a}", "", true},
		{"named-schemeless-host", "x/{a}", "//{a}", "", true},
		{"named-host-suffix", "x/{a}", "https://example.com{a}/", "", true},
		{"bad-shortcode", "gh/*/*", "https://example.com/*/*", "", true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := NormalizeWildcardURL(tt.input, tt.shortcode)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("NormalizeWildcardURL(%q, %q) = %q, want error", tt.input, tt.shortcode, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeWildcardURL(%q, %q) unexpected error: %v", tt.input, tt.shortcode, err)
			}
			if got != tt.want {
				t.Errorf("NormalizeWildcardURL(%q, %q) = %q, want %q", tt.input, tt.shortcode, got, tt.want)
			}
		})
	}
}

// End to end through the domain: the link from George's screenshot.
func TestNamedPattern_GitHubPR(t *testing.T) {
	t.Parallel()
	dest, err := NormalizeWildcardURL("https://github.com/intrinsic-org/{repo}/pull/{pr}", "gh/{repo}/{pr}")
	if err != nil {
		t.Fatal(err)
	}
	link := &Link{Shortcode: "gh/{repo}/{pr}", URL: dest}
	m, caps, ok := ResolveWildcard([]*Link{link}, "gh/intrinsic/10")
	if !ok || m != link {
		t.Fatalf("no match: %v %v", caps, ok)
	}
	got, err := SubstituteWildcard(m.URL, caps)
	if err != nil || got != "https://github.com/intrinsic-org/intrinsic/pull/10" {
		t.Errorf("got %q, %v", got, err)
	}
}

func TestHasPlaceholder(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]bool{
		"https://ex.com/{a}":                           true,
		"https://ex.com/*":                             true,
		`https://grafana.ex/explore?left={"q":1}`:      false,
		"https://ex.com/{A}":                           false,
		"https://ex.com/plain":                         false,
		`https://kibana.ex/app#/?_g=(time:(from:now))`: false,
	} {
		if got := HasPlaceholder(in); got != want {
			t.Errorf("HasPlaceholder(%q) = %v, want %v", in, got, want)
		}
	}
}
