package domain

import (
	"errors"
	"testing"
)

func TestValidShortcode(t *testing.T) {
	t.Parallel()
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ValidShortcode(tt.input)
			if got != tt.want {
				t.Errorf("ValidShortcode(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, input, want string
		wantErr           bool
	}{
		{"already-https", "https://example.com", "https://example.com", false},
		{"already-http", "http://example.com", "http://example.com", false},
		{"no-scheme", "example.com", "https://example.com", false},
		{"whitespace", "  example.com  ", "https://example.com", false},
		{"with-path", "example.com/foo", "https://example.com/foo", false},

		// Open-redirect / phishing vectors that NormalizeURL must reject.
		{"empty", "", "", true},
		{"only-whitespace", "   ", "", true},
		{"javascript-scheme", "javascript:alert(1)", "", true},
		{"data-scheme", "data:text/html,<script>alert(1)</script>", "", true},
		{"file-scheme", "file:///etc/passwd", "", true},
		{"protocol-relative", "//evil.example.com", "", true},
		{"embedded-credentials", "http://trusted@evil.com", "", true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := NormalizeURL(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("NormalizeURL(%q) = %q, want error", tt.input, got)
				}
				if !errors.Is(err, ErrInvalidURL) {
					t.Errorf("NormalizeURL(%q) error = %v, want ErrInvalidURL", tt.input, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeURL(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("NormalizeURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
