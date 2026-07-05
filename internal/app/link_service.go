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
// A shortcode containing "*" is a wildcard/parameterized link: it must be a
// valid wildcard pattern (e.g. "pulls/*") and its destination must contain a
// matching "*". A "*" on one side but not the other is rejected as a mismatch.
func (s *linkService) CreateLink(shortcode, rawURL, description, owner string) (*domain.Link, error) {
	shortcode = domain.NormalizeShortcode(shortcode)
	isWildcard := domain.IsWildcardShortcode(shortcode)
	if isWildcard {
		if !domain.ValidWildcardShortcode(shortcode) {
			return nil, errors.New("invalid wildcard shortcode: use letters, numbers, hyphens, and underscores in each segment with a single trailing '*' (e.g. pulls/*)")
		}
	} else if !domain.ValidShortcode(shortcode) {
		return nil, errors.New("invalid shortcode: use only letters, numbers, hyphens, and underscores")
	}
	if strings.TrimSpace(rawURL) == "" {
		return nil, errors.New("url is required")
	}
	if isWildcard != strings.Contains(rawURL, "*") {
		return nil, errors.New("wildcard mismatch: a '*' shortcode requires exactly one '*' in the destination, and vice-versa")
	}
	normalized, err := s.normalizeDestination(rawURL, isWildcard)
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

// normalizeDestination validates and normalizes a destination URL, using the
// wildcard-aware path when the link is parameterized.
func (s *linkService) normalizeDestination(rawURL string, isWildcard bool) (string, error) {
	if isWildcard {
		normalized, err := domain.NormalizeWildcardURL(rawURL)
		if err != nil {
			return "", fmt.Errorf("invalid wildcard url: destination must be a valid http(s) URL with exactly one '*' in the path (not the host), e.g. https://example.com/pull/*")
		}
		return normalized, nil
	}
	normalized, err := domain.NormalizeURL(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid url: must be a valid http or https URL")
	}
	return normalized, nil
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
	if rawURL != "" {
		normalized, err := domain.NormalizeURL(rawURL)
		if err != nil {
			return nil, fmt.Errorf("invalid url: must be a valid http or https URL")
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
// no exact (non-pattern) match do we fall back to wildcard links, capturing a
// single path segment and substituting it into the destination. Click counts
// for wildcard hits are recorded against the pattern link.
func (s *linkService) RedirectLink(shortcode string) (*domain.Link, error) {
	// 1. Exact match. A stored pattern link ("pulls/*") visited by its literal
	//    text must NOT resolve as exact — fall through to wildcard handling
	//    (which will reject the "*" capture and 404).
	link, err := s.repo.GetLink(shortcode)
	switch {
	case err == nil && !domain.IsWildcardShortcode(link.Shortcode):
		s.incrementClicks(shortcode)
		return link, nil
	case err != nil && err != domain.ErrNotFound:
		return nil, err
	}

	// 2. Wildcard fallback.
	links, err := s.repo.ListLinks()
	if err != nil {
		return nil, err
	}
	match, capture, ok := domain.ResolveWildcard(links, shortcode)
	if !ok {
		return nil, domain.ErrNotFound
	}
	finalURL, err := domain.SubstituteWildcard(match.URL, capture)
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
