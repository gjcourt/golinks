package domain

import (
	"errors"
	"net/url"
	"strings"
)

// Wildcard (parameterized) links let a single link capture one path segment
// and substitute it into the destination. A wildcard shortcode contains
// exactly one "*" as its final path segment (e.g. "pulls/*"); the destination
// URL contains exactly one "*" (in the path or query — never the host or
// scheme) which is replaced by the captured, percent-encoded segment.
//
//	shortcode  "pulls/*"
//	dest       "https://github.com/gjcourt/homelab/pull/*"
//	go/pulls/10  ->  https://github.com/gjcourt/homelab/pull/10
//
// Security: the captured value is restricted to a safe charset
// (letters/digits/-/_/.), rejected otherwise, and percent-escaped before
// substitution. Because it can never contain "/", ":", "?", "#" it cannot
// break out of its path segment, alter the scheme/host, or inject a second
// URL. The destination is validated through NormalizeURL at creation with the
// "*" held out of the host, so the redirect target's origin is fixed.

// ErrInvalidCapture is returned when a captured wildcard segment fails the
// safe-charset check, or when a stored template is malformed.
var ErrInvalidCapture = errors.New("invalid wildcard capture")

// wildcardMarker is the single wildcard character permitted in shortcodes and
// destination templates.
const wildcardMarker = "*"

// urlWildcardSentinel is a placeholder substituted for "*" while a wildcard
// destination is run through NormalizeURL, so the URL parser validates the
// real scheme/host. It is plain ASCII letters so url.Parse never escapes it
// and its presence in the parsed host reliably signals that the "*" sat in
// the host position (which we reject).
const urlWildcardSentinel = "golinkswildcardsentinel"

// IsWildcardShortcode reports whether s uses wildcard syntax (contains "*").
// It does not validate the pattern — use ValidWildcardShortcode for that.
func IsWildcardShortcode(s string) bool {
	return strings.Contains(s, wildcardMarker)
}

// ValidWildcardShortcode reports whether s is a well-formed wildcard pattern:
// one or more literal segments — each matching the plain shortcode charset —
// followed by a final "*" segment, with exactly one "*" overall. A bare "*"
// (no literal prefix) is rejected so a wildcard can never match every path.
//
//	pulls/*      valid
//	jira/*       valid
//	a/b/*        valid
//	*            invalid (no literal prefix)
//	pulls/*/x    invalid ("*" not final)
//	pu*ls/*      invalid (two "*")
func ValidWildcardShortcode(s string) bool {
	if len(s) < 2 || len(s) > 100 {
		return false
	}
	if strings.Count(s, wildcardMarker) != 1 {
		return false
	}
	segments := strings.Split(s, "/")
	if len(segments) < 2 {
		return false
	}
	if segments[len(segments)-1] != wildcardMarker {
		return false
	}
	for _, seg := range segments[:len(segments)-1] {
		if !shortcodeRe.MatchString(seg) {
			return false
		}
	}
	return true
}

// wildcardPrefix returns the literal portion of a wildcard shortcode — the
// text up to and including the final "/". For "pulls/*" this is "pulls/".
func wildcardPrefix(shortcode string) string {
	return strings.TrimSuffix(shortcode, wildcardMarker)
}

// MatchWildcard tests whether path matches the wildcard shortcode and, if so,
// returns the captured segment. The capture is a single path segment: it must
// be non-empty, contain no "/", and satisfy the safe-charset rule. Multi-
// segment inputs (e.g. "pulls/10/extra") do not match a single-"*" pattern.
func MatchWildcard(shortcode, path string) (capture string, ok bool) {
	if !ValidWildcardShortcode(shortcode) {
		return "", false
	}
	prefix := wildcardPrefix(shortcode)
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}
	capture = path[len(prefix):]
	if !validCapture(capture) {
		return "", false
	}
	return capture, true
}

// validCapture enforces the safe-charset rule for a captured segment:
// non-empty, letters/digits/-/_/. only. This rejects "/", "%", spaces, ":",
// "?", "#", and anything resembling a nested URL or traversal ("../").
func validCapture(capture string) bool {
	if capture == "" {
		return false
	}
	// Reject the dot segments outright: even though they carry no "/", "." and
	// ".." would let a capture walk the destination's path (same-origin, but
	// still not the intended target).
	if capture == "." || capture == ".." {
		return false
	}
	for _, r := range capture {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_' || r == '.':
		default:
			return false
		}
	}
	return true
}

// ResolveWildcard finds the most specific wildcard link matching path and
// returns it with the captured segment. Precedence: the longest literal
// prefix wins (most specific). Ties are broken by the lexicographically
// smallest shortcode so resolution is deterministic regardless of storage
// order. Non-wildcard or malformed links are ignored.
func ResolveWildcard(links []*Link, path string) (match *Link, capture string, ok bool) {
	bestPrefixLen := -1
	for _, l := range links {
		if !ValidWildcardShortcode(l.Shortcode) {
			continue
		}
		cap, matched := MatchWildcard(l.Shortcode, path)
		if !matched {
			continue
		}
		prefixLen := len(wildcardPrefix(l.Shortcode))
		switch {
		case prefixLen > bestPrefixLen:
		case prefixLen == bestPrefixLen && match != nil && l.Shortcode < match.Shortcode:
		default:
			continue
		}
		bestPrefixLen = prefixLen
		match = l
		capture = cap
		ok = true
	}
	return match, capture, ok
}

// SubstituteWildcard fills the single "*" in a stored destination template
// with the percent-escaped capture. The capture is re-validated (defence in
// depth) and PathEscaped; because the safe charset is a subset of the URL
// unreserved set, escaping is effectively identity but guarantees the value
// stays confined to one path segment.
func SubstituteWildcard(template, capture string) (string, error) {
	if !validCapture(capture) {
		return "", ErrInvalidCapture
	}
	if strings.Count(template, wildcardMarker) != 1 {
		return "", ErrInvalidCapture
	}
	escaped := url.PathEscape(capture)
	return strings.Replace(template, wildcardMarker, escaped, 1), nil
}

// NormalizeWildcardURL validates and normalizes a wildcard destination
// template. The destination must contain exactly one "*", and that "*" must
// sit in the path or query — not the scheme or host. It returns the
// normalized template (scheme/host canonicalized, "*" preserved) suitable for
// SubstituteWildcard at resolution time.
func NormalizeWildcardURL(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if strings.Count(s, wildcardMarker) != 1 {
		return "", ErrInvalidURL
	}
	// Hold the "*" out of parsing by swapping in a plain-ASCII sentinel, so
	// NormalizeURL validates the real scheme/host. If the input already
	// contains the sentinel the post-normalize count check below rejects it.
	withSentinel := strings.Replace(s, wildcardMarker, urlWildcardSentinel, 1)
	normalized, err := NormalizeURL(withSentinel)
	if err != nil {
		return "", err
	}
	u, err := url.Parse(normalized)
	if err != nil {
		return "", ErrInvalidURL
	}
	// A "*" in the host (e.g. "https://*.example.com" or a bare "*") would
	// let the wildcard change the redirect origin — reject it.
	if strings.Contains(strings.ToLower(u.Host), urlWildcardSentinel) {
		return "", ErrInvalidURL
	}
	if strings.Count(normalized, urlWildcardSentinel) != 1 {
		return "", ErrInvalidURL
	}
	return strings.Replace(normalized, urlWildcardSentinel, wildcardMarker, 1), nil
}
