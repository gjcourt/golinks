package memory

import (
	"sync"
	"time"

	"github.com/george/golinks/internal/domain"
)

// Repository implements domain.LinkRepository and domain.UserRepository in memory.
type Repository struct {
	mu          sync.RWMutex
	links       map[string]*domain.Link
	users       map[string]*domain.User
	lastClicked map[string]time.Time
}

// NewRepository creates a new in-memory repository.
func NewRepository() *Repository {
	return &Repository{
		links:       make(map[string]*domain.Link),
		users:       make(map[string]*domain.User),
		lastClicked: make(map[string]time.Time),
	}
}

// CreateLink stores a new link.
func (r *Repository) CreateLink(link *domain.Link) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.links[link.Shortcode]; exists {
		return domain.ErrAlreadyExists
	}

	// Clone to avoid external mutation
	newLink := *link
	newLink.ID = int64(len(r.links) + 1)
	newLink.CreatedAt = time.Now()
	newLink.UpdatedAt = time.Now()

	r.links[link.Shortcode] = &newLink
	return nil
}

// GetLink retrieves a link by shortcode.
func (r *Repository) GetLink(shortcode string) (*domain.Link, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	link, exists := r.links[shortcode]
	if !exists {
		return nil, domain.ErrNotFound
	}

	// Return a copy
	l := *link
	return &l, nil
}

// UpdateLink updates an existing link.
func (r *Repository) UpdateLink(shortcode string, link *domain.Link) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.links[shortcode]
	if !exists {
		return domain.ErrNotFound
	}

	// Update fields
	existing.URL = link.URL
	existing.Description = link.Description
	existing.UpdatedAt = time.Now()

	return nil
}

// DeleteLink removes a link.
func (r *Repository) DeleteLink(shortcode string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.links[shortcode]; !exists {
		return domain.ErrNotFound
	}

	delete(r.links, shortcode)
	delete(r.lastClicked, shortcode)
	return nil
}

// ListLinks returns all links.
func (r *Repository) ListLinks() ([]*domain.Link, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	links := make([]*domain.Link, 0, len(r.links))
	for _, l := range r.links {
		// Return copy
		cp := *l
		links = append(links, &cp)
	}
	return links, nil
}

// IncrementClickCount increments the click count for a link.
func (r *Repository) IncrementClickCount(shortcode string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	link, exists := r.links[shortcode]
	if !exists {
		return domain.ErrNotFound
	}

	link.ClickCount++
	r.lastClicked[shortcode] = time.Now()

	return nil
}

// GetStats retrieves statistics for a link.
func (r *Repository) GetStats(shortcode string) (*domain.LinkStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	link, exists := r.links[shortcode]
	if !exists {
		return nil, domain.ErrNotFound
	}

	return &domain.LinkStats{
		Shortcode:   shortcode,
		ClickCount:  link.ClickCount,
		LastClicked: r.lastClicked[shortcode],
	}, nil
}

// Close is a no-op for in-memory repository.
func (r *Repository) Close() error {
	return nil
}

// CreateUser creates a new user.
func (r *Repository) CreateUser(user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[user.Username]; exists {
		return domain.ErrAlreadyExists
	}

	newUser := *user
	newUser.ID = int64(len(r.users) + 1)
	newUser.CreatedAt = time.Now()

	r.users[user.Username] = &newUser
	return nil
}

// GetUserByUsername retrieves a user by username.
func (r *Repository) GetUserByUsername(username string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.users[username]
	if !exists {
		return nil, domain.ErrNotFound
	}

	// Return copy
	u := *user
	return &u, nil
}

// CountUsers returns the total number of users.
func (r *Repository) CountUsers() (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return int64(len(r.users)), nil
}
