package domain

import (
	"regexp"
	"strings"
)

var shortcodeRe = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// ValidShortcode checks whether a shortcode is syntactically valid.
func ValidShortcode(s string) bool {
	return len(s) >= 1 && len(s) <= 100 && shortcodeRe.MatchString(s)
}

// NormalizeURL ensures the URL has an http(s) scheme.
func NormalizeURL(raw string) string {
	u := strings.TrimSpace(raw)
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		u = "https://" + u
	}
	return u
}
