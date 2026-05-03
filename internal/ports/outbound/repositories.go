// Package outbound holds the driven-port interfaces for persistent storage.
package outbound

import "github.com/george/golinks/internal/domain"

// LinkRepository is the driven port for persisting and retrieving links.
type LinkRepository interface {
	CreateLink(link *domain.Link) error
	GetLink(shortcode string) (*domain.Link, error)
	UpdateLink(shortcode string, link *domain.Link) error
	DeleteLink(shortcode string) error
	ListLinks() ([]*domain.Link, error)
	IncrementClickCount(shortcode string) error
	GetStats(shortcode string) (*domain.LinkStats, error)
	Close() error
}

// UserRepository is the driven port for persisting and retrieving users.
type UserRepository interface {
	CreateUser(user *domain.User) error
	GetUserByUsername(username string) (*domain.User, error)
	CountUsers() (int64, error)
	Close() error
}
