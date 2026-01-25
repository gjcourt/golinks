package models

import "time"

// Link represents a go link with its metadata
type Link struct {
	ID          int64     `json:"id"`
	Shortcode   string    `json:"shortcode"`
	URL         string    `json:"url"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	ClickCount  int64     `json:"click_count"`
}

// LinkStats contains statistics for a link
type LinkStats struct {
	Shortcode   string    `json:"shortcode"`
	ClickCount  int64     `json:"click_count"`
	LastClicked time.Time `json:"last_clicked,omitempty"`
}

// CreateLinkRequest is the request body for creating a link
type CreateLinkRequest struct {
	Shortcode   string `json:"shortcode"`
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// UpdateLinkRequest is the request body for updating a link
type UpdateLinkRequest struct {
	URL         string `json:"url,omitempty"`
	Description string `json:"description,omitempty"`
}
