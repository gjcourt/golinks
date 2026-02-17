package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var shortcodeRe = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// linkService implements LinkService.
type linkService struct {
	repo LinkRepository
}

// NewLinkService creates a LinkService backed by the given repository.
func NewLinkService(repo LinkRepository) LinkService {
	return &linkService{repo: repo}
}

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

// CreateLink validates inputs and persists a new link owned by owner.
func (s *linkService) CreateLink(shortcode, url, description, owner string) (*Link, error) {
	shortcode = strings.TrimSpace(shortcode)
	if !ValidShortcode(shortcode) {
		return nil, errors.New("invalid shortcode: use only letters, numbers, hyphens, and underscores")
	}
	url = NormalizeURL(url)
	if strings.TrimSpace(url) == "" || url == "https://" {
		return nil, errors.New("url is required")
	}
	link := &Link{
		Shortcode:   shortcode,
		URL:         url,
		Description: strings.TrimSpace(description),
		Owner:       owner,
	}
	if err := s.repo.CreateLink(link); err != nil {
		return nil, err
	}
	return link, nil
}

// GetLink retrieves a link by shortcode.
func (s *linkService) GetLink(shortcode string) (*Link, error) {
	return s.repo.GetLink(shortcode)
}

// UpdateLink patches the URL and/or description of an existing link.
// Only the owner or an admin may update a link.
func (s *linkService) UpdateLink(shortcode, url, description, username string, isAdmin bool) (*Link, error) {
	existing, err := s.repo.GetLink(shortcode)
	if err != nil {
		return nil, err
	}
	if !isAdmin && existing.Owner != username {
		return nil, fmt.Errorf("%w: only the owner or an admin can update this link", ErrForbidden)
	}
	if url != "" {
		existing.URL = NormalizeURL(url)
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
// Only the owner or an admin may delete a link.
func (s *linkService) DeleteLink(shortcode, username string, isAdmin bool) error {
	existing, err := s.repo.GetLink(shortcode)
	if err != nil {
		return err
	}
	if !isAdmin && existing.Owner != username {
		return fmt.Errorf("%w: only the owner or an admin can delete this link", ErrForbidden)
	}
	return s.repo.DeleteLink(shortcode)
}

// ListLinks returns all links.
func (s *linkService) ListLinks() ([]*Link, error) {
	return s.repo.ListLinks()
}

// RedirectLink fetches a link and asynchronously increments its click count.
func (s *linkService) RedirectLink(shortcode string) (*Link, error) {
	link, err := s.repo.GetLink(shortcode)
	if err != nil {
		return nil, err
	}
	go func() {
		_ = s.repo.IncrementClickCount(shortcode)
	}()
	return link, nil
}

// GetLinkStats returns click statistics for a link.
func (s *linkService) GetLinkStats(shortcode string) (*LinkStats, error) {
	return s.repo.GetStats(shortcode)
}
