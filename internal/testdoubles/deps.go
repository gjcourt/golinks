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

func (f *FakeLinkRepository) CreateLink(link *domain.Link) error {
	if f.CreateLinkFn != nil {
		return f.CreateLinkFn(link)
	}
	return nil
}

func (f *FakeLinkRepository) GetLink(shortcode string) (*domain.Link, error) {
	if f.GetLinkFn != nil {
		return f.GetLinkFn(shortcode)
	}
	return nil, domain.ErrNotFound
}

func (f *FakeLinkRepository) UpdateLink(shortcode string, link *domain.Link) error {
	if f.UpdateLinkFn != nil {
		return f.UpdateLinkFn(shortcode, link)
	}
	return nil
}

func (f *FakeLinkRepository) DeleteLink(shortcode string) error {
	if f.DeleteLinkFn != nil {
		return f.DeleteLinkFn(shortcode)
	}
	return nil
}

func (f *FakeLinkRepository) ListLinks() ([]*domain.Link, error) {
	if f.ListLinksFn != nil {
		return f.ListLinksFn()
	}
	return nil, nil
}

func (f *FakeLinkRepository) IncrementClickCount(shortcode string) error {
	if f.IncrementClickCountFn != nil {
		return f.IncrementClickCountFn(shortcode)
	}
	return nil
}

func (f *FakeLinkRepository) GetStats(shortcode string) (*domain.LinkStats, error) {
	if f.GetStatsFn != nil {
		return f.GetStatsFn(shortcode)
	}
	return nil, domain.ErrNotFound
}

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

func (f *FakeUserRepository) CreateUser(user *domain.User) error {
	if f.CreateUserFn != nil {
		return f.CreateUserFn(user)
	}
	return nil
}

func (f *FakeUserRepository) GetUserByUsername(username string) (*domain.User, error) {
	if f.GetUserByUsernameFn != nil {
		return f.GetUserByUsernameFn(username)
	}
	return nil, domain.ErrNotFound
}

func (f *FakeUserRepository) CountUsers() (int64, error) {
	if f.CountUsersFn != nil {
		return f.CountUsersFn()
	}
	return 0, nil
}

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
