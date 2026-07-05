package domain

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

var shortcodeRe = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// ErrInvalidURL is returned by NormalizeURL when the input cannot be turned
// into a syntactically valid http(s) URL with a non-empty host.
var ErrInvalidURL = errors.New("invalid url")

// ValidShortcode checks whether a shortcode is syntactically valid.
func ValidShortcode(s string) bool {
	return len(s) >= 1 && len(s) <= 100 && shortcodeRe.MatchString(s)
}

// NormalizeShortcode strips input conveniences before validation: surrounding
// whitespace, a leading "go/", and a stray leading "/". Users routinely type
// the shortcode exactly as it's displayed (e.g. "go/ms"); without this that
// becomes an invalid shortcode (the "/" fails ValidShortcode). Codes that
// merely start with "go" (e.g. "google") are untouched because the trimmed
// prefix includes the slash.
func NormalizeShortcode(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "go/")
	s = strings.TrimPrefix(s, "/")
	return s
}

// NormalizeURL ensures the URL parses as an http(s) URL with a non-empty
// host. If the input lacks a scheme it is treated as https. Inputs that do
// not parse, that have a non-http(s) scheme, or that resolve to an empty
// host (e.g. "javascript:alert(1)", "//evil.example.com", schemeless inputs
// containing only a path) return ErrInvalidURL — preventing the redirect
// handler from being abused as an open redirector.
func NormalizeURL(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", ErrInvalidURL
	}

	// Reject protocol-relative inputs outright — they parse with an empty
	// scheme and a host attacker-controlled, which would silently become
	// "https:////evil.example.com" if we naively prefixed.
	if strings.HasPrefix(s, "//") {
		return "", ErrInvalidURL
	}

	// If the input has no scheme, assume https. We detect "no scheme" by
	// looking for "://" rather than `url.Parse`'s scheme detection because
	// inputs like "example.com/path" parse with an empty scheme and the
	// host buried inside Path.
	if !strings.Contains(s, "://") {
		s = "https://" + s
	}

	u, err := url.Parse(s)
	if err != nil {
		return "", ErrInvalidURL
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", ErrInvalidURL
	}
	if u.Host == "" {
		return "", ErrInvalidURL
	}
	// Reject embedded credentials — a common phishing vector
	// ("http://trusted@evil.com/").
	if u.User != nil {
		return "", ErrInvalidURL
	}
	return u.String(), nil
}
