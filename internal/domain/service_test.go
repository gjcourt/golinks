package domain

import "testing"

func TestValidShortcode(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"simple", "docs", true},
		{"with-hyphen", "my-link", true},
		{"with-underscore", "my_link", true},
		{"alphanumeric", "abc123", true},
		{"empty", "", false},
		{"spaces", "has space", false},
		{"special-chars", "no/slash", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidShortcode(tt.input)
			if got != tt.want {
				t.Errorf("ValidShortcode(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name, input, want string
	}{
		{"already-https", "https://example.com", "https://example.com"},
		{"already-http", "http://example.com", "http://example.com"},
		{"no-scheme", "example.com", "https://example.com"},
		{"whitespace", "  example.com  ", "https://example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeURL(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
