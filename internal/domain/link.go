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
	ErrForbidden     = errors.New("forbidden")
)

// Link represents a go-link with its metadata.
type Link struct {
	ID          int64
	Shortcode   string
	URL         string
	Description string
	Owner       string
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

// User represents a registered golinks user.
type User struct {
	ID           int64
	Username     string
	PasswordHash string
	Role         string // "admin" or "user"
	CreatedAt    time.Time
}

// Session represents an active user session.
type Session struct {
	Token     string
	UserID    int64
	ExpiresAt time.Time
	CreatedAt time.Time
}
