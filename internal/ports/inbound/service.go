// Package inbound holds the driving-port interfaces exposed to HTTP adapters.
package inbound

import "github.com/george/golinks/internal/domain"

// LinkService encapsulates all business rules for link management.
type LinkService interface {
	CreateLink(shortcode, url, description, owner string) (*domain.Link, error)
	GetLink(shortcode string) (*domain.Link, error)
	UpdateLink(shortcode, url, description, username string, isAdmin bool) (*domain.Link, error)
	DeleteLink(shortcode, username string, isAdmin bool) error
	ListLinks() ([]*domain.Link, error)
	RedirectLink(shortcode string) (*domain.Link, error)
	GetLinkStats(shortcode string) (*domain.LinkStats, error)
}
