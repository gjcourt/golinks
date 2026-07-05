package domain

import "testing"

func TestValidWildcardShortcode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, input string
		want        bool
	}{
		{"simple", "pulls/*", true},
		{"jira", "jira/*", true},
		{"multi-literal", "gh/pr/*", true},
		{"hyphen-underscore-segments", "my-repo/pull_requests/*", true},
		{"bare-star-rejected", "*", false},
		{"star-not-final", "pulls/*/x", false},
		{"two-stars", "pu*ls/*", false},
		{"no-star", "pulls/x", false},
		{"trailing-slash-no-star", "pulls/", false},
		{"star-mid-segment", "pull*/*", false},
		{"empty-literal-segment", "pulls//*", false},
		{"bad-char-in-literal", "pu lls/*", false},
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
		name, shortcode, path, wantCapture string
		wantOK                             bool
	}{
		{"basic", "pulls/*", "pulls/10", "10", true},
		{"dotted-capture", "release/*", "release/v1.2.3", "v1.2.3", true},
		{"hyphen-capture", "u/*", "u/john-doe", "john-doe", true},
		{"multi-segment-no-match", "pulls/*", "pulls/10/extra", "", false},
		{"trailing-slash-no-match", "pulls/*", "pulls/10/", "", false},
		{"empty-capture-no-match", "pulls/*", "pulls/", "", false},
		{"prefix-mismatch", "pulls/*", "issues/10", "", false},
		{"dotdot-rejected", "pulls/*", "pulls/..", "", false},
		{"dot-rejected", "pulls/*", "pulls/.", "", false},
		{"encoded-percent-rejected", "pulls/*", "pulls/%2e%2e", "", false},
		{"space-rejected", "pulls/*", "pulls/a b", "", false},
		{"colon-rejected", "pulls/*", "pulls/http:", "", false},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotCapture, gotOK := MatchWildcard(tt.shortcode, tt.path)
			if gotOK != tt.wantOK || gotCapture != tt.wantCapture {
				t.Errorf("MatchWildcard(%q, %q) = (%q, %v), want (%q, %v)",
					tt.shortcode, tt.path, gotCapture, gotOK, tt.wantCapture, tt.wantOK)
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
	match, capture, ok := ResolveWildcard(links, "gh/pr/123")
	if !ok {
		t.Fatalf("expected a match")
	}
	if match.Shortcode != "gh/pr/*" || capture != "123" {
		t.Errorf("got (%q, %q), want (gh/pr/*, 123)", match.Shortcode, capture)
	}

	// A single-segment tail matches only the shorter pattern.
	match, capture, ok = ResolveWildcard(links, "gh/xyz")
	if !ok || match.Shortcode != "gh/*" || capture != "xyz" {
		t.Errorf("got (%v, %q, %v), want (gh/*, xyz, true)", match, capture, ok)
	}
}

func TestResolveWildcard_OrderIndependent(t *testing.T) {
	t.Parallel()
	// Selection must not depend on slice order: the more-specific (longer
	// literal prefix) pattern wins whichever way the links are listed.
	short := &Link{Shortcode: "gh/*", URL: "https://example.com/a/*"}
	long := &Link{Shortcode: "gh/pr/*", URL: "https://example.com/b/*"}
	for _, order := range [][]*Link{{short, long}, {long, short}} {
		m, capture, ok := ResolveWildcard(order, "gh/pr/9")
		if !ok || m.Shortcode != "gh/pr/*" || capture != "9" {
			t.Errorf("order %v -> (%v, %q, %v), want (gh/pr/*, 9, true)", order, m, capture, ok)
		}
	}
}

func TestResolveWildcard_NoMatch(t *testing.T) {
	t.Parallel()
	links := []*Link{
		{Shortcode: "pulls/*", URL: "https://example.com/pull/*"},
		{Shortcode: "docs", URL: "https://example.com/docs"}, // non-wildcard, ignored
	}
	if m, _, ok := ResolveWildcard(links, "issues/1"); ok {
		t.Errorf("issues/1 unexpectedly matched %v", m)
	}
	// A pattern visited by its own literal text does not resolve (the "*"
	// capture fails the safe-charset rule).
	if m, _, ok := ResolveWildcard(links, "pulls/*"); ok {
		t.Errorf("literal pattern path matched %v", m)
	}
}

func TestSubstituteWildcard(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, template, capture, want string
		wantErr                       bool
	}{
		{"basic", "https://github.com/gjcourt/homelab/pull/*", "10", "https://github.com/gjcourt/homelab/pull/10", false},
		{"dotted", "https://example.com/r/*", "v1.2.3", "https://example.com/r/v1.2.3", false},
		{"unsafe-slash", "https://example.com/*", "a/b", "", true},
		{"unsafe-percent", "https://example.com/*", "%2e%2e", "", true},
		{"unsafe-colon", "https://example.com/*", "http:", "", true},
		{"empty-capture", "https://example.com/*", "", "", true},
		{"template-no-star", "https://example.com/x", "10", "", true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := SubstituteWildcard(tt.template, tt.capture)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("SubstituteWildcard(%q, %q) = %q, want error", tt.template, tt.capture, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("SubstituteWildcard(%q, %q) = %q, want %q", tt.template, tt.capture, got, tt.want)
			}
		})
	}
}

func TestNormalizeWildcardURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, input, want string
		wantErr           bool
	}{
		{"path-star", "https://github.com/gjcourt/homelab/pull/*", "https://github.com/gjcourt/homelab/pull/*", false},
		{"no-scheme", "example.com/pull/*", "https://example.com/pull/*", false},
		{"query-star", "https://example.com/search?q=*", "https://example.com/search?q=*", false},

		// Rejections.
		{"no-star", "https://example.com/pull/x", "", true},
		{"two-stars", "https://example.com/*/*", "", true},
		{"star-in-host", "https://*.example.com/x", "", true},
		{"bare-star", "*", "", true},
		{"star-is-host", "https://*", "", true},
		{"javascript-scheme", "javascript:*", "", true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := NormalizeWildcardURL(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("NormalizeWildcardURL(%q) = %q, want error", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeWildcardURL(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("NormalizeWildcardURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
