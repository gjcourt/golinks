package domain

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Mock repository — in-memory, used by all service tests
// ---------------------------------------------------------------------------

type mockRepo struct {
	mu     sync.Mutex
	links  map[string]*Link
	nextID int64
}

func newMockRepo() *mockRepo {
	return &mockRepo{links: make(map[string]*Link), nextID: 1}
}

func (m *mockRepo) CreateLink(link *Link) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.links[link.Shortcode]; ok {
		return ErrAlreadyExists
	}
	now := time.Now()
	link.ID = m.nextID
	link.CreatedAt = now
	link.UpdatedAt = now
	m.nextID++
	stored := *link
	m.links[link.Shortcode] = &stored
	return nil
}

func (m *mockRepo) GetLink(shortcode string) (*Link, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	l, ok := m.links[shortcode]
	if !ok {
		return nil, ErrNotFound
	}
	copy := *l
	return &copy, nil
}

func (m *mockRepo) UpdateLink(shortcode string, link *Link) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.links[shortcode]; !ok {
		return ErrNotFound
	}
	link.UpdatedAt = time.Now()
	stored := *link
	m.links[shortcode] = &stored
	return nil
}

func (m *mockRepo) DeleteLink(shortcode string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.links[shortcode]; !ok {
		return ErrNotFound
	}
	delete(m.links, shortcode)
	return nil
}

func (m *mockRepo) ListLinks() ([]*Link, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*Link, 0, len(m.links))
	for _, l := range m.links {
		copy := *l
		out = append(out, &copy)
	}
	return out, nil
}

func (m *mockRepo) IncrementClickCount(shortcode string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	l, ok := m.links[shortcode]
	if !ok {
		return ErrNotFound
	}
	l.ClickCount++
	return nil
}

func (m *mockRepo) GetStats(shortcode string) (*LinkStats, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	l, ok := m.links[shortcode]
	if !ok {
		return nil, ErrNotFound
	}
	return &LinkStats{Shortcode: l.Shortcode, ClickCount: l.ClickCount}, nil
}

func (m *mockRepo) Close() error { return nil }

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestValidShortcode(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"simple", "docs", true},
		{"with-hyphen", "my-link", true},
		{"with-underscore", "my_link", true},
		{"alphanumeric", "abc123", true},
		{"empty", "", false},
		{"spaces", "has space", false},
		{"special-chars", "no/slash", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidShortcode(tt.input)
			if got != tt.want {
				t.Errorf("ValidShortcode(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name, input, want string
	}{
		{"already-https", "https://example.com", "https://example.com"},
		{"already-http", "http://example.com", "http://example.com"},
		{"no-scheme", "example.com", "https://example.com"},
		{"whitespace", "  example.com  ", "https://example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeURL(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCreateLink(t *testing.T) {
	tests := []struct {
		name, shortcode, url, desc, wantErr string
	}{
		{"happy-path", "docs", "https://docs.example.com", "Documentation", ""},
		{"no-scheme", "wiki", "wiki.example.com", "", ""},
		{"invalid-shortcode", "bad code", "https://x.com", "", "invalid shortcode"},
		{"empty-shortcode", "", "https://x.com", "", "invalid shortcode"},
		{"empty-url", "ok", "", "", "url is required"},
		{"whitespace-url", "ok2", "   ", "", "url is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewLinkService(newMockRepo())
			link, err := svc.CreateLink(tt.shortcode, tt.url, tt.desc, "alice")
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if link.Shortcode != tt.shortcode {
				t.Errorf("shortcode = %q, want %q", link.Shortcode, tt.shortcode)
			}
			if link.Owner != "alice" {
				t.Errorf("owner = %q, want %q", link.Owner, "alice")
			}
		})
	}
}

func TestCreateLink_Duplicate(t *testing.T) {
	repo := newMockRepo()
	svc := NewLinkService(repo)
	if _, err := svc.CreateLink("docs", "https://a.com", "", "alice"); err != nil {
		t.Fatalf("first create: %v", err)
	}
	_, err := svc.CreateLink("docs", "https://b.com", "", "alice")
	if err != ErrAlreadyExists {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestUpdateLink(t *testing.T) {
	repo := newMockRepo()
	svc := NewLinkService(repo)
	if _, err := svc.CreateLink("docs", "https://old.com", "old desc", "alice"); err != nil {
		t.Fatalf("create: %v", err)
	}

	tests := []struct {
		name, shortcode, url, desc string
		wantErr                    error
	}{
		{"update-url", "docs", "https://new.com", "", nil},
		{"update-desc", "docs", "", "new desc", nil},
		{"not-found", "nope", "https://x.com", "", ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.UpdateLink(tt.shortcode, tt.url, tt.desc, "alice", false)
			if err != tt.wantErr {
				t.Errorf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdateLink_Authorization(t *testing.T) {
	repo := newMockRepo()
	svc := NewLinkService(repo)
	if _, err := svc.CreateLink("docs", "https://old.com", "desc", "alice"); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Non-owner should be forbidden
	_, err := svc.UpdateLink("docs", "https://new.com", "", "bob", false)
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}

	// Admin non-owner should succeed
	_, err = svc.UpdateLink("docs", "https://admin.com", "", "bob", true)
	if err != nil {
		t.Errorf("admin update: unexpected error: %v", err)
	}
}

func TestDeleteLink(t *testing.T) {
	repo := newMockRepo()
	svc := NewLinkService(repo)
	_, _ = svc.CreateLink("docs", "https://a.com", "", "alice")

	tests := []struct {
		name, shortcode string
		wantErr         error
	}{
		{"existing", "docs", nil},
		{"already-deleted", "docs", ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := svc.DeleteLink(tt.shortcode, "alice", false); err != tt.wantErr {
				t.Errorf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeleteLink_Authorization(t *testing.T) {
	repo := newMockRepo()
	svc := NewLinkService(repo)
	_, _ = svc.CreateLink("docs", "https://a.com", "", "alice")

	// Non-owner should be forbidden
	err := svc.DeleteLink("docs", "bob", false)
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}

	// Admin non-owner should succeed
	err = svc.DeleteLink("docs", "bob", true)
	if err != nil {
		t.Errorf("admin delete: unexpected error: %v", err)
	}
}

func TestListLinks(t *testing.T) {
	svc := NewLinkService(newMockRepo())
	links, _ := svc.ListLinks()
	if len(links) != 0 {
		t.Fatalf("expected 0 links, got %d", len(links))
	}
	_, _ = svc.CreateLink("a", "https://a.com", "", "alice")
	_, _ = svc.CreateLink("b", "https://b.com", "", "bob")
	links, _ = svc.ListLinks()
	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(links))
	}
}

func TestRedirectLink(t *testing.T) {
	repo := newMockRepo()
	svc := NewLinkService(repo)
	if _, err := svc.CreateLink("docs", "https://docs.example.com", "", "alice"); err != nil {
		t.Fatalf("create: %v", err)
	}

	link, err := svc.RedirectLink("docs")
	if err != nil {
		t.Fatalf("redirect: %v", err)
	}
	if link.URL != "https://docs.example.com" {
		t.Errorf("url = %q, want %q", link.URL, "https://docs.example.com")
	}

	if _, err := svc.RedirectLink("nope"); err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}

	// give goroutine a moment to increment
	time.Sleep(50 * time.Millisecond)
	stats, _ := svc.GetLinkStats("docs")
	if stats.ClickCount < 1 {
		t.Errorf("click_count = %d, expected >= 1", stats.ClickCount)
	}
}

func TestGetLinkStats_NotFound(t *testing.T) {
	svc := NewLinkService(newMockRepo())
	_, err := svc.GetLinkStats("nope")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
