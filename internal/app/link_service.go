// Package app holds the application services (use-case layer).
package app

import (
	"errors"
	"fmt"
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
func (s *linkService) CreateLink(shortcode, url, description, owner string) (*domain.Link, error) {
	shortcode = strings.TrimSpace(shortcode)
	if !domain.ValidShortcode(shortcode) {
		return nil, errors.New("invalid shortcode: use only letters, numbers, hyphens, and underscores")
	}
	url = domain.NormalizeURL(url)
	if strings.TrimSpace(url) == "" || url == "https://" {
		return nil, errors.New("url is required")
	}
	link := &domain.Link{
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
func (s *linkService) GetLink(shortcode string) (*domain.Link, error) {
	return s.repo.GetLink(shortcode)
}

// UpdateLink patches the URL and/or description of an existing link.
func (s *linkService) UpdateLink(shortcode, url, description, username string, isAdmin bool) (*domain.Link, error) {
	existing, err := s.repo.GetLink(shortcode)
	if err != nil {
		return nil, err
	}
	if !isAdmin && existing.Owner != username {
		return nil, fmt.Errorf("%w: only the owner or an admin can update this link", domain.ErrForbidden)
	}
	if url != "" {
		existing.URL = domain.NormalizeURL(url)
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

// RedirectLink fetches a link and asynchronously increments its click count.
func (s *linkService) RedirectLink(shortcode string) (*domain.Link, error) {
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
func (s *linkService) GetLinkStats(shortcode string) (*domain.LinkStats, error) {
	return s.repo.GetStats(shortcode)
}
