// Package domain holds the core business logic.
package domain

import (
	"errors"
	"time"
)

// Domain errors — storage adapters must return these sentinel values.
var (
	ErrNotFound      = errors.New("link not found")
	ErrAlreadyExists = errors.New("link already exists")
)

// Link represents a go-link with its metadata.
type Link struct {
	ID          int64
	Shortcode   string
	URL         string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ClickCount  int64
}

// LinkStats contains click statistics for a link.
type LinkStats struct {
	Shortcode   string
	ClickCount  int64
	LastClicked time.Time
}
