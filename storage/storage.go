package storage

import "github.com/george/golinks/models"

// Store defines the interface for link storage
type Store interface {
	// CreateLink creates a new link
	CreateLink(link *models.Link) error

	// GetLink retrieves a link by shortcode
	GetLink(shortcode string) (*models.Link, error)

	// UpdateLink updates an existing link
	UpdateLink(shortcode string, link *models.Link) error

	// DeleteLink deletes a link by shortcode
	DeleteLink(shortcode string) error

	// ListLinks returns all links
	ListLinks() ([]*models.Link, error)

	// IncrementClickCount increments the click count for a link
	IncrementClickCount(shortcode string) error

	// GetStats returns statistics for a link
	GetStats(shortcode string) (*models.LinkStats, error)

	// Close closes the storage connection
	Close() error
}
