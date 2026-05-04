// Package testdoubles provides function-field fakes for outbound ports,
// used by unit tests in higher layers to avoid spinning up real adapters.
package testdoubles

import (
	"github.com/george/golinks/internal/domain"
	"github.com/george/golinks/internal/ports/outbound"
)

// FakeLinkRepository is a function-field fake for outbound.LinkRepository.
type FakeLinkRepository struct {
	CreateLinkFn          func(link *domain.Link) error
	GetLinkFn             func(shortcode string) (*domain.Link, error)
	UpdateLinkFn          func(shortcode string, link *domain.Link) error
	DeleteLinkFn          func(shortcode string) error
	ListLinksFn           func() ([]*domain.Link, error)
	IncrementClickCountFn func(shortcode string) error
	GetStatsFn            func(shortcode string) (*domain.LinkStats, error)
	CloseFn               func() error
}

var _ outbound.LinkRepository = (*FakeLinkRepository)(nil)

// CreateLink delegates to CreateLinkFn if set, otherwise returns nil.
func (f *FakeLinkRepository) CreateLink(link *domain.Link) error {
	if f.CreateLinkFn != nil {
		return f.CreateLinkFn(link)
	}
	return nil
}

// GetLink delegates to GetLinkFn if set, otherwise returns domain.ErrNotFound.
func (f *FakeLinkRepository) GetLink(shortcode string) (*domain.Link, error) {
	if f.GetLinkFn != nil {
		return f.GetLinkFn(shortcode)
	}
	return nil, domain.ErrNotFound
}

// UpdateLink delegates to UpdateLinkFn if set, otherwise returns nil.
func (f *FakeLinkRepository) UpdateLink(shortcode string, link *domain.Link) error {
	if f.UpdateLinkFn != nil {
		return f.UpdateLinkFn(shortcode, link)
	}
	return nil
}

// DeleteLink delegates to DeleteLinkFn if set, otherwise returns nil.
func (f *FakeLinkRepository) DeleteLink(shortcode string) error {
	if f.DeleteLinkFn != nil {
		return f.DeleteLinkFn(shortcode)
	}
	return nil
}

// ListLinks delegates to ListLinksFn if set, otherwise returns (nil, nil).
func (f *FakeLinkRepository) ListLinks() ([]*domain.Link, error) {
	if f.ListLinksFn != nil {
		return f.ListLinksFn()
	}
	return nil, nil
}

// IncrementClickCount delegates to IncrementClickCountFn if set, otherwise returns nil.
func (f *FakeLinkRepository) IncrementClickCount(shortcode string) error {
	if f.IncrementClickCountFn != nil {
		return f.IncrementClickCountFn(shortcode)
	}
	return nil
}

// GetStats delegates to GetStatsFn if set, otherwise returns domain.ErrNotFound.
func (f *FakeLinkRepository) GetStats(shortcode string) (*domain.LinkStats, error) {
	if f.GetStatsFn != nil {
		return f.GetStatsFn(shortcode)
	}
	return nil, domain.ErrNotFound
}

// Close delegates to CloseFn if set, otherwise returns nil.
func (f *FakeLinkRepository) Close() error {
	if f.CloseFn != nil {
		return f.CloseFn()
	}
	return nil
}

// FakeUserRepository is a function-field fake for outbound.UserRepository.
type FakeUserRepository struct {
	CreateUserFn        func(user *domain.User) error
	GetUserByUsernameFn func(username string) (*domain.User, error)
	CountUsersFn        func() (int64, error)
	CloseFn             func() error
}

var _ outbound.UserRepository = (*FakeUserRepository)(nil)

// CreateUser delegates to CreateUserFn if set, otherwise returns nil.
func (f *FakeUserRepository) CreateUser(user *domain.User) error {
	if f.CreateUserFn != nil {
		return f.CreateUserFn(user)
	}
	return nil
}

// GetUserByUsername delegates to GetUserByUsernameFn if set, otherwise returns domain.ErrNotFound.
func (f *FakeUserRepository) GetUserByUsername(username string) (*domain.User, error) {
	if f.GetUserByUsernameFn != nil {
		return f.GetUserByUsernameFn(username)
	}
	return nil, domain.ErrNotFound
}

// CountUsers delegates to CountUsersFn if set, otherwise returns (0, nil).
func (f *FakeUserRepository) CountUsers() (int64, error) {
	if f.CountUsersFn != nil {
		return f.CountUsersFn()
	}
	return 0, nil
}

// Close delegates to CloseFn if set, otherwise returns nil.
func (f *FakeUserRepository) Close() error {
	if f.CloseFn != nil {
		return f.CloseFn()
	}
	return nil
}

// ServerDeps aggregates all outbound-port fakes for unit tests.
type ServerDeps struct {
	Links *FakeLinkRepository
	Users *FakeUserRepository
}

// NewServerDeps returns a ServerDeps with all fakes initialised to safe zero-value defaults.
func NewServerDeps() *ServerDeps {
	return &ServerDeps{
		Links: &FakeLinkRepository{},
		Users: &FakeUserRepository{},
	}
}
