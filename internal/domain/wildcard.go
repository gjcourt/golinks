package domain

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

// Pattern (parameterized) links capture path segments and substitute them
// into the destination. A pattern shortcode is one or more "/"-separated
// segments, each either a literal or a whole-segment parameter:
//
//	named     "gh/{repo}/{pr}"   dest "https://github.com/acme/{repo}/pull/{pr}"
//	          go/gh/api/10  ->  https://github.com/acme/api/pull/10
//	legacy    "pulls/*"          dest "https://github.com/acme/homelab/pull/*"
//	          go/pulls/10   ->  https://github.com/acme/homelab/pull/10
//
// Named parameters ({name}: lowercase letters, digits, "_"; starting with a
// letter) may appear in any segment after the first, in any number, with
// literals between them. Every parameter must appear in the destination at
// least once and the destination may use no other; a parameter may be used
// more than once. The legacy form is a single trailing "*" with exactly one
// "*" in the destination; it behaves as one anonymous parameter. The two
// forms can't be mixed in one link.
//
// The first segment is always a literal, so no pattern can match every path.
//
// Security: each captured value is restricted to a safe charset
// (letters/digits/-/_/.), rejected otherwise, and percent-escaped before
// substitution. Because it can never contain "/", ":", "?", "#" it cannot
// break out of its path segment, alter the scheme/host, or inject a second
// URL. The destination is validated through NormalizeURL at creation with
// every placeholder held out of the host, so the redirect target's origin is
// fixed.

// ErrInvalidCapture is returned when a captured segment fails the
// safe-charset check, or when a stored template is malformed.
var ErrInvalidCapture = errors.New("invalid wildcard capture")

// ErrInvalidPattern is returned for a malformed pattern shortcode, or a
// destination whose placeholders don't match the shortcode's parameters.
var ErrInvalidPattern = errors.New("invalid pattern")

// wildcardMarker is the legacy anonymous parameter.
const wildcardMarker = "*"

// anonymousParam is the parameter name the legacy "*" is stored under.
const anonymousParam = "*"

var (
	paramSegmentRe = regexp.MustCompile(`^\{([a-z][a-z0-9_]{0,31})\}$`)
	// placeholderRe finds destination placeholders: {name} or the legacy *.
	placeholderRe = regexp.MustCompile(`\{([a-z][a-z0-9_]{0,31})\}|\*`)
)

// urlParamSentinel prefixes the placeholders substituted for each parameter
// while a destination is run through NormalizeURL, so the URL parser
// validates the real scheme/host. It is plain lowercase ASCII so url.Parse
// never escapes it, and its presence in the parsed host reliably signals a
// parameter in the host position (which we reject).
const urlParamSentinel = "golinksparamsentinel"

// patternSegment is one "/"-separated part of a pattern shortcode.
type patternSegment struct {
	literal string // set for a literal segment
	param   string // set for a parameter segment ("*" for the legacy form)
}

// pattern is a parsed pattern shortcode.
type pattern struct {
	segments []patternSegment
	params   []string // in order of appearance
	literals int      // number of literal segments
	litLen   int      // total length of literal segments
}

// IsWildcardShortcode reports whether s uses pattern syntax ("*" or "{…}").
// It does not validate the pattern — use ValidWildcardShortcode for that.
func IsWildcardShortcode(s string) bool {
	return strings.ContainsAny(s, "*{}")
}

// ValidWildcardShortcode reports whether s is a well-formed pattern.
//
//	pulls/*            valid (legacy)
//	gh/{repo}/{pr}     valid
//	gh/{repo}/pull/{n} valid (literal between parameters)
//	*  {x}/y           invalid (first segment must be literal)
//	pulls/*/x          invalid ("*" must be the final segment)
//	gh/*/*             invalid (use named parameters for more than one)
//	gh/{a}/{a}         invalid (parameter repeated)
//	gh/{a}/*           invalid (forms mixed)
//	gh/x{a}            invalid (a parameter is a whole segment)
func ValidWildcardShortcode(s string) bool {
	_, err := parsePattern(s)
	return err == nil
}

// PatternParams returns the parameter names of a pattern shortcode, in order
// ("*" for the legacy form).
func PatternParams(s string) ([]string, error) {
	p, err := parsePattern(s)
	if err != nil {
		return nil, err
	}
	return p.params, nil
}

func parsePattern(s string) (pattern, error) {
	if len(s) < 2 || len(s) > 100 || !IsWildcardShortcode(s) {
		return pattern{}, ErrInvalidPattern
	}
	parts := strings.Split(s, "/")
	if len(parts) < 2 {
		return pattern{}, ErrInvalidPattern
	}
	var p pattern
	seen := map[string]bool{}
	for i, part := range parts {
		if err := p.addSegment(part, i, len(parts), seen); err != nil {
			return pattern{}, err
		}
	}
	if len(p.params) == 0 {
		return pattern{}, ErrInvalidPattern
	}
	return p, nil
}

// addSegment appends the i-th of n shortcode segments to p. seen holds the
// parameter names so far (and "*" once the legacy form is used).
func (p *pattern) addSegment(part string, i, n int, seen map[string]bool) error {
	switch {
	case shortcodeRe.MatchString(part):
		p.segments = append(p.segments, patternSegment{literal: part})
		p.literals++
		p.litLen += len(part)
		return nil
	case i == 0:
		return ErrInvalidPattern // first segment must be literal
	case part == wildcardMarker:
		if len(p.params) > 0 || i != n-1 {
			return ErrInvalidPattern // single trailing "*", not mixed
		}
		seen[anonymousParam] = true
		p.segments = append(p.segments, patternSegment{param: anonymousParam})
		p.params = append(p.params, anonymousParam)
		return nil
	}
	m := paramSegmentRe.FindStringSubmatch(part)
	if m == nil || seen[anonymousParam] || seen[m[1]] {
		return ErrInvalidPattern
	}
	seen[m[1]] = true
	p.segments = append(p.segments, patternSegment{param: m[1]})
	p.params = append(p.params, m[1])
	return nil
}

// match tests path against the pattern and returns the captures.
func (p pattern) match(path string) (map[string]string, bool) {
	parts := strings.Split(path, "/")
	if len(parts) != len(p.segments) {
		return nil, false
	}
	captures := make(map[string]string, len(p.params))
	for i, seg := range p.segments {
		if seg.param == "" {
			if parts[i] != seg.literal {
				return nil, false
			}
			continue
		}
		if !validCapture(parts[i]) {
			return nil, false
		}
		captures[seg.param] = parts[i]
	}
	return captures, true
}

// MatchWildcard tests whether path matches the pattern shortcode and, if so,
// returns the captured segments keyed by parameter name ("*" for the legacy
// form). Each capture is exactly one path segment and must satisfy the
// safe-charset rule; the path must have exactly as many segments as the
// pattern.
func MatchWildcard(shortcode, path string) (captures map[string]string, ok bool) {
	p, err := parsePattern(shortcode)
	if err != nil {
		return nil, false
	}
	return p.match(path)
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

// ResolveWildcard finds the most specific pattern link matching path and
// returns it with its captures. Precedence: more literal segments wins, then
// more literal characters, then the lexicographically smallest shortcode, so
// resolution is deterministic regardless of storage order. Non-pattern or
// malformed links are ignored.
func ResolveWildcard(links []*Link, path string) (match *Link, captures map[string]string, ok bool) {
	var best pattern
	for _, l := range links {
		p, err := parsePattern(l.Shortcode)
		if err != nil {
			continue
		}
		caps, matched := p.match(path)
		if !matched {
			continue
		}
		if match != nil {
			switch {
			case p.literals != best.literals:
				if p.literals < best.literals {
					continue
				}
			case p.litLen != best.litLen:
				if p.litLen < best.litLen {
					continue
				}
			case l.Shortcode >= match.Shortcode:
				continue
			}
		}
		best, match, captures, ok = p, l, caps, true
	}
	return match, captures, ok
}

// SubstituteWildcard fills every placeholder in a stored destination
// template with its percent-escaped capture. Each capture is re-validated
// (defence in depth) and PathEscaped; because the safe charset is a subset of
// the URL unreserved set, escaping is effectively identity but guarantees the
// value stays confined to one path segment. A placeholder with no capture is
// an error — never redirect to a half-substituted target.
//
// A legacy "*" template substitutes only its "*": a literal "{…}" it carries
// (e.g. in a query string) is left alone, as before named parameters.
//
// The result must keep the template's scheme and host: the host is checked
// here, at every redirect, so a template that somehow lets a capture reach
// the host (a row written before creation-time validation, or directly to
// the store) is refused rather than turned into an open redirect.
func SubstituteWildcard(template string, captures map[string]string) (string, error) {
	for _, v := range captures {
		if !validCapture(v) {
			return "", ErrInvalidCapture
		}
	}
	_, legacy := captures[anonymousParam]
	missing := false
	fill := func(value func(name string) (string, bool)) string {
		return placeholderRe.ReplaceAllStringFunc(template, func(ph string) string {
			if legacy != (ph == wildcardMarker) {
				return ph // the other form's syntax: literal text
			}
			name := anonymousParam
			if !legacy {
				name = ph[1 : len(ph)-1]
			}
			v, ok := value(name)
			if !ok {
				missing = true
				return ph
			}
			return v
		})
	}
	out := fill(func(name string) (string, bool) {
		v, ok := captures[name]
		return url.PathEscape(v), ok
	})
	if missing {
		return "", ErrInvalidCapture
	}
	// The same template with every placeholder set to a fixed, harmless
	// value shows where the host really is.
	probe := fill(func(string) (string, bool) { return "x", true })
	if !sameOrigin(out, probe) {
		return "", ErrInvalidCapture
	}
	return out, nil
}

// sameOrigin reports whether a and b parse as http(s) URLs with the same
// scheme and a non-empty, equal host (port included).
func sameOrigin(a, b string) bool {
	ua, errA := url.Parse(a)
	ub, errB := url.Parse(b)
	if errA != nil || errB != nil || ua.Host == "" || ua.User != nil || ub.User != nil {
		return false
	}
	scheme := strings.ToLower(ua.Scheme)
	return (scheme == "http" || scheme == "https") &&
		strings.EqualFold(ua.Scheme, ub.Scheme) && strings.EqualFold(ua.Host, ub.Host)
}

// HasPlaceholder reports whether a destination contains pattern syntax — a
// "{name}" or the legacy "*". Other braces (e.g. JSON in a query string, as
// Grafana and Kibana links carry) are not placeholders.
func HasPlaceholder(dest string) bool {
	return placeholderRe.MatchString(dest)
}

// NormalizeWildcardURL validates and normalizes a destination template for
// the pattern shortcode. The destination's placeholders must be exactly the
// shortcode's parameters (each used at least once, no others; for the legacy
// form, exactly one "*"), and none may sit in the scheme or host. Stray "{" or
// "}" are rejected. It returns the normalized template (scheme/host
// canonicalized, placeholders preserved) for SubstituteWildcard.
func NormalizeWildcardURL(raw, shortcode string) (string, error) {
	p, err := parsePattern(shortcode)
	if err != nil {
		return "", err
	}
	s := strings.TrimSpace(raw)
	if strings.Contains(s, urlParamSentinel) {
		return "", ErrInvalidURL
	}
	withSentinels, placeholders, err := sentinelize(s, p.params)
	if err != nil {
		return "", err
	}
	normalized, err := NormalizeURL(withSentinels)
	if err != nil {
		return "", err
	}
	if err := checkSentinels(normalized, len(placeholders)); err != nil {
		return "", err
	}
	for i, ph := range placeholders {
		normalized = strings.Replace(normalized, urlParamSentinel+sentinelSuffix(i), ph, 1)
	}
	return normalized, nil
}

// sentinelize swaps each destination placeholder for a distinct plain-ASCII
// sentinel so the URL can be parsed, and checks the placeholders are exactly
// params: each used at least once, no others, no stray "{", "}" or "*"; the
// legacy form allows exactly one "*". It returns the placeholders in order of
// appearance, for restoring.
func sentinelize(s string, params []string) (string, []string, error) {
	want := map[string]bool{}
	for _, name := range params {
		want[name] = true
	}
	used := map[string]bool{}
	var placeholders []string
	unknown := false
	// A legacy template's only placeholder is "*": any "{…}" in it is
	// literal text (e.g. JSON in a query string), exactly as SubstituteWildcard
	// treats it, and isn't checked for stray braces.
	legacy := want[anonymousParam]
	out := placeholderRe.ReplaceAllStringFunc(s, func(ph string) string {
		if legacy && ph != wildcardMarker {
			return ph
		}
		name := anonymousParam
		if ph != wildcardMarker {
			name = ph[1 : len(ph)-1]
		}
		unknown = unknown || !want[name]
		used[name] = true
		placeholders = append(placeholders, ph)
		return urlParamSentinel + sentinelSuffix(len(placeholders)-1)
	})
	switch {
	case unknown, len(used) != len(want):
		return "", nil, ErrInvalidPattern
	case legacy && (len(placeholders) != 1 || strings.Contains(out, "*")):
		return "", nil, ErrInvalidPattern // legacy form: exactly one "*"
	case !legacy && strings.ContainsAny(out, "{}*"):
		return "", nil, ErrInvalidPattern // stray brace or "*" in a named template
	}
	return out, placeholders, nil
}

// checkSentinels verifies a normalized destination still carries all n
// sentinels and none sits in the host — a parameter in the host (e.g.
// "https://{org}.example.com") would let a capture change the redirect
// origin.
func checkSentinels(normalized string, n int) error {
	u, err := url.Parse(normalized)
	if err != nil {
		return ErrInvalidURL
	}
	if strings.Contains(strings.ToLower(u.Host), urlParamSentinel) {
		return ErrInvalidURL
	}
	if strings.Count(normalized, urlParamSentinel) != n {
		return ErrInvalidURL
	}
	return nil
}

// sentinelSuffix returns a fixed-width, letters-only suffix for the i-th
// placeholder: every sentinel is distinct and none is a prefix of another, so
// all survive URL normalization unchanged and restore unambiguously. (Past
// 26^3 placeholders suffixes repeat; restoring replaces the first occurrence
// in order, so the result is still correct.)
func sentinelSuffix(i int) string {
	return string([]rune{rune('a' + i/676%26), rune('a' + i/26%26), rune('a' + i%26)})
}
