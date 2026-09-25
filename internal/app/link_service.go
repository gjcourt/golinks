// Package app holds the application services (use-case layer).
package app

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/george/golinks/internal/domain"
	"github.com/george/golinks/internal/ports/inbound"
	"github.com/george/golinks/internal/ports/outbound"
)

type linkService struct {
	repo outbound.LinkRepository
}

// NewLinkService creates a LinkService backed by the given repository.
func NewLinkService(repo outbound.LinkRepository) inbound.LinkService {
	return &linkService{repo: repo}
}

// CreateLink validates inputs and persists a new link owned by owner.
//
// A shortcode with "{name}" segments (or the legacy trailing "*") is a
// pattern link: it must be a valid pattern (e.g. "gh/{repo}/{pr}" or
// "pulls/*") and its destination must use exactly its parameters. Pattern
// syntax on one side but not the other is rejected.
func (s *linkService) CreateLink(shortcode, rawURL, description, owner string) (*domain.Link, error) {
	shortcode = domain.NormalizeShortcode(shortcode)
	isPattern := domain.IsWildcardShortcode(shortcode)
	if isPattern {
		if !domain.ValidWildcardShortcode(shortcode) {
			return nil, errors.New(invalidPatternMsg)
		}
	} else if !domain.ValidShortcode(shortcode) {
		return nil, errors.New("invalid shortcode: use only letters, numbers, hyphens, and underscores")
	}
	if strings.TrimSpace(rawURL) == "" {
		return nil, errors.New("url is required")
	}
	if !isPattern && hasPlaceholder(rawURL) {
		return nil, errors.New("the destination has {parameters} or '*' but the shortcode has none — add them to the shortcode, e.g. gh/{repo}/{pr}")
	}
	normalized, err := s.normalizeDestination(rawURL, shortcode, isPattern)
	if err != nil {
		return nil, err
	}
	link := &domain.Link{
		Shortcode:   shortcode,
		URL:         normalized,
		Description: strings.TrimSpace(description),
		Owner:       owner,
	}
	if err := s.repo.CreateLink(link); err != nil {
		return nil, err
	}
	return link, nil
}

const invalidPatternMsg = "invalid pattern shortcode: start with a plain segment, then use {name} segments " +
	"(lowercase letters, digits, _) for each part to capture, e.g. gh/{repo}/{pr} — " +
	"or a single trailing '*', e.g. pulls/*"

// hasPlaceholder reports whether a destination contains pattern syntax — a
// "{name}" or the legacy "*". Braces that aren't placeholders (JSON in a
// query string) are fine in a plain link.
func hasPlaceholder(rawURL string) bool {
	return domain.HasPlaceholder(rawURL)
}

// normalizeDestination validates and normalizes a destination URL; for a
// pattern link it must use exactly the shortcode's parameters.
func (s *linkService) normalizeDestination(rawURL, shortcode string, isPattern bool) (string, error) {
	if isPattern {
		normalized, err := domain.NormalizeWildcardURL(rawURL, shortcode)
		if err != nil {
			params, _ := domain.PatternParams(shortcode)
			return "", fmt.Errorf("invalid pattern url: destination must be a valid http(s) URL that uses %s "+
				"in its path or query (not the host), e.g. https://github.com/acme/{repo}/pull/{pr}", describeParams(params))
		}
		return normalized, nil
	}
	normalized, err := domain.NormalizeURL(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid url: must be a valid http or https URL")
	}
	return normalized, nil
}

// describeParams renders a shortcode's parameters for an error message.
func describeParams(params []string) string {
	if len(params) == 1 && params[0] == "*" {
		return "exactly one '*'"
	}
	parts := make([]string, len(params))
	for i, p := range params {
		parts[i] = "{" + p + "}"
	}
	return "each of " + strings.Join(parts, ", ") + " (and no others)"
}

// GetLink retrieves a link by shortcode.
func (s *linkService) GetLink(shortcode string) (*domain.Link, error) {
	return s.repo.GetLink(shortcode)
}

// UpdateLink patches the URL and/or description of an existing link.
func (s *linkService) UpdateLink(shortcode, rawURL, description, username string, isAdmin bool) (*domain.Link, error) {
	existing, err := s.repo.GetLink(shortcode)
	if err != nil {
		return nil, err
	}
	if !isAdmin && existing.Owner != username {
		return nil, fmt.Errorf("%w: only the owner or an admin can update this link", domain.ErrForbidden)
	}
	// An edit that resends the destination unchanged (the admin UI always
	// sends it, even for a description-only edit) doesn't re-validate it:
	// rows accepted by older rules stay editable.
	if rawURL != "" && rawURL != existing.URL {
		// A pattern link's destination must keep using its parameters, so it
		// goes through the same validation as at creation.
		isPattern := domain.IsWildcardShortcode(existing.Shortcode)
		if !isPattern && hasPlaceholder(rawURL) {
			return nil, errors.New("the destination has {parameters} or '*' but this link's shortcode has none")
		}
		normalized, err := s.normalizeDestination(rawURL, existing.Shortcode, isPattern)
		if err != nil {
			return nil, err
		}
		existing.URL = normalized
	}
	if description != "" {
		existing.Description = strings.TrimSpace(description)
	}
	if err := s.repo.UpdateLink(shortcode, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

// DeleteLink removes a link.
func (s *linkService) DeleteLink(shortcode, username string, isAdmin bool) error {
	existing, err := s.repo.GetLink(shortcode)
	if err != nil {
		return err
	}
	if !isAdmin && existing.Owner != username {
		return fmt.Errorf("%w: only the owner or an admin can delete this link", domain.ErrForbidden)
	}
	return s.repo.DeleteLink(shortcode)
}

// ListLinks returns all links.
func (s *linkService) ListLinks() ([]*domain.Link, error) {
	return s.repo.ListLinks()
}

// RedirectLink fetches a link and increments its click count synchronously.
//
// The increment is intentionally synchronous (rather than spawned in a
// goroutine) so that errors are surfaced via the logger and so that
// in-flight increments are not silently lost on shutdown. Latency cost is
// a single UPDATE on the same connection that already served the GET.
// Resolution order: an exact shortcode match always wins. Only when there is
// no exact (non-pattern) match do we fall back to pattern links, capturing
// each parameter's path segment and substituting it into the destination.
// Click counts for pattern hits are recorded against the pattern link.
func (s *linkService) RedirectLink(shortcode string) (*domain.Link, error) {
	// 1. Exact match. A stored pattern link ("pulls/*", "gh/{repo}") visited
	//    by its literal text must NOT resolve as exact — fall through to
	//    pattern handling (which rejects the "*"/"{…}" capture and 404s).
	link, err := s.repo.GetLink(shortcode)
	switch {
	case err == nil && !domain.IsWildcardShortcode(link.Shortcode):
		s.incrementClicks(shortcode)
		return link, nil
	case err != nil && err != domain.ErrNotFound:
		return nil, err
	}

	// 2. Pattern fallback.
	links, err := s.repo.ListLinks()
	if err != nil {
		return nil, err
	}
	match, captures, ok := domain.ResolveWildcard(links, shortcode)
	if !ok {
		return nil, domain.ErrNotFound
	}
	finalURL, err := domain.SubstituteWildcard(match.URL, captures)
	if err != nil {
		// Capture failed the safe-charset check — treat as a miss, never
		// redirect to a half-substituted or unsafe target.
		return nil, domain.ErrNotFound
	}
	s.incrementClicks(match.Shortcode)
	resolved := *match
	resolved.URL = finalURL
	return &resolved, nil
}

// incrementClicks bumps the click counter, logging (not failing) on error.
func (s *linkService) incrementClicks(shortcode string) {
	if err := s.repo.IncrementClickCount(shortcode); err != nil {
		slog.Warn("increment click count failed", "shortcode", shortcode, "err", err)
	}
}

// GetLinkStats returns click statistics for a link.
func (s *linkService) GetLinkStats(shortcode string) (*domain.LinkStats, error) {
	return s.repo.GetStats(shortcode)
}
