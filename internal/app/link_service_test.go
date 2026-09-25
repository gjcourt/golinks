package app_test

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/george/golinks/internal/app"
	"github.com/george/golinks/internal/domain"
)

// mockRepo is an in-memory LinkRepository used by all service tests.
type mockRepo struct {
	mu     sync.Mutex
	links  map[string]*domain.Link
	nextID int64
}

func newMockRepo() *mockRepo {
	return &mockRepo{links: make(map[string]*domain.Link), nextID: 1}
}

func (m *mockRepo) CreateLink(link *domain.Link) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.links[link.Shortcode]; ok {
		return domain.ErrAlreadyExists
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

func (m *mockRepo) GetLink(shortcode string) (*domain.Link, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	l, ok := m.links[shortcode]
	if !ok {
		return nil, domain.ErrNotFound
	}
	copy := *l
	return &copy, nil
}

func (m *mockRepo) UpdateLink(shortcode string, link *domain.Link) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.links[shortcode]; !ok {
		return domain.ErrNotFound
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
		return domain.ErrNotFound
	}
	delete(m.links, shortcode)
	return nil
}

func (m *mockRepo) ListLinks() ([]*domain.Link, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*domain.Link, 0, len(m.links))
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
		return domain.ErrNotFound
	}
	l.ClickCount++
	return nil
}

func (m *mockRepo) GetStats(shortcode string) (*domain.LinkStats, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	l, ok := m.links[shortcode]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &domain.LinkStats{Shortcode: l.Shortcode, ClickCount: l.ClickCount}, nil
}

func (m *mockRepo) Close() error { return nil }

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
			svc := app.NewLinkService(newMockRepo())
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

func TestCreateLink_NormalizesGoPrefix(t *testing.T) {
	svc := app.NewLinkService(newMockRepo())
	link, err := svc.CreateLink("go/ms", "https://grafana.example.com", "MQTT Scope", "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if link.Shortcode != "ms" {
		t.Errorf("shortcode = %q, want %q (leading go/ should be stripped)", link.Shortcode, "ms")
	}
}

func TestCreateLink_Duplicate(t *testing.T) {
	repo := newMockRepo()
	svc := app.NewLinkService(repo)
	if _, err := svc.CreateLink("docs", "https://a.com", "", "alice"); err != nil {
		t.Fatalf("first create: %v", err)
	}
	_, err := svc.CreateLink("docs", "https://b.com", "", "alice")
	if err != domain.ErrAlreadyExists {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestUpdateLink(t *testing.T) {
	repo := newMockRepo()
	svc := app.NewLinkService(repo)
	if _, err := svc.CreateLink("docs", "https://old.com", "old desc", "alice"); err != nil {
		t.Fatalf("create: %v", err)
	}

	tests := []struct {
		name, shortcode, url, desc string
		wantErr                    error
	}{
		{"update-url", "docs", "https://new.com", "", nil},
		{"update-desc", "docs", "", "new desc", nil},
		{"not-found", "nope", "https://x.com", "", domain.ErrNotFound},
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
	svc := app.NewLinkService(repo)
	if _, err := svc.CreateLink("docs", "https://old.com", "desc", "alice"); err != nil {
		t.Fatalf("create: %v", err)
	}

	_, err := svc.UpdateLink("docs", "https://new.com", "", "bob", false)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}

	_, err = svc.UpdateLink("docs", "https://admin.com", "", "bob", true)
	if err != nil {
		t.Errorf("admin update: unexpected error: %v", err)
	}
}

func TestDeleteLink(t *testing.T) {
	repo := newMockRepo()
	svc := app.NewLinkService(repo)
	_, _ = svc.CreateLink("docs", "https://a.com", "", "alice")

	tests := []struct {
		name, shortcode string
		wantErr         error
	}{
		{"existing", "docs", nil},
		{"already-deleted", "docs", domain.ErrNotFound},
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
	svc := app.NewLinkService(repo)
	_, _ = svc.CreateLink("docs", "https://a.com", "", "alice")

	err := svc.DeleteLink("docs", "bob", false)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}

	err = svc.DeleteLink("docs", "bob", true)
	if err != nil {
		t.Errorf("admin delete: unexpected error: %v", err)
	}
}

func TestListLinks(t *testing.T) {
	svc := app.NewLinkService(newMockRepo())
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
	svc := app.NewLinkService(repo)
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

	if _, err := svc.RedirectLink("nope"); err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	stats, _ := svc.GetLinkStats("docs")
	if stats.ClickCount < 1 {
		t.Errorf("click_count = %d, expected >= 1", stats.ClickCount)
	}
}

func TestGetLinkStats_NotFound(t *testing.T) {
	svc := app.NewLinkService(newMockRepo())
	_, err := svc.GetLinkStats("nope")
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Wildcard / parameterized links
// ---------------------------------------------------------------------------

func TestCreateLink_Wildcard(t *testing.T) {
	tests := []struct {
		name, shortcode, url, wantShortcode, wantErr string
	}{
		{"valid-pair", "pulls/*", "https://github.com/gjcourt/homelab/pull/*", "pulls/*", ""},
		{"normalizes-go-prefix", "go/pulls/*", "https://example.com/pull/*", "pulls/*", ""},
		{"no-scheme-dest", "jira/*", "jira.example.com/browse/*", "jira/*", ""},
		{"shortcode-star-dest-none", "pulls/*", "https://example.com/pull/x", "", "invalid pattern url"},
		{"dest-star-shortcode-none", "pulls", "https://example.com/pull/*", "", "shortcode has none"},
		{"two-stars-shortcode", "a/*/*", "https://example.com/*", "", "invalid pattern shortcode"},
		{"bare-star-shortcode", "*", "https://example.com/*", "", "invalid pattern shortcode"},
		{"star-in-host", "h/*", "https://*.example.com/x", "", "invalid pattern url"},
		{"two-stars-dest", "p/*", "https://example.com/*/*", "", "invalid pattern url"},

		// Named parameters.
		{"named-pair", "gh/{repo}/{pr}", "https://github.com/intrinsic-org/{repo}/pull/{pr}", "gh/{repo}/{pr}", ""},
		{"named-go-prefix", "go/gh/{repo}/{pr}", "https://github.com/acme/{repo}/pull/{pr}", "gh/{repo}/{pr}", ""},
		{"named-dest-missing-param", "gh/{repo}/{pr}", "https://github.com/acme/{repo}", "", "each of {repo}, {pr}"},
		{"named-dest-unknown-param", "gh/{repo}", "https://github.com/acme/{repo}/{pr}", "", "invalid pattern url"},
		{"named-dest-params-shortcode-none", "gh", "https://github.com/acme/{repo}", "", "shortcode has none"},
		{"named-param-in-host", "o/{org}", "https://{org}.example.com/", "", "invalid pattern url"},
		{"named-first-segment", "{repo}/x", "https://example.com/{repo}", "", "invalid pattern shortcode"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := app.NewLinkService(newMockRepo())
			link, err := svc.CreateLink(tt.shortcode, tt.url, "", "alice")
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if link.Shortcode != tt.wantShortcode {
				t.Errorf("shortcode = %q, want %q", link.Shortcode, tt.wantShortcode)
			}
		})
	}
}

func TestRedirectLink_Wildcard(t *testing.T) {
	repo := newMockRepo()
	svc := app.NewLinkService(repo)
	if _, err := svc.CreateLink("pulls/*", "https://github.com/gjcourt/homelab/pull/*", "", "alice"); err != nil {
		t.Fatalf("create: %v", err)
	}

	tests := []struct {
		name, path, wantURL string
		wantErr             error
	}{
		{"basic", "pulls/10", "https://github.com/gjcourt/homelab/pull/10", nil},
		{"dotted", "pulls/v1.2.3", "https://github.com/gjcourt/homelab/pull/v1.2.3", nil},
		{"multi-segment-404", "pulls/10/extra", "", domain.ErrNotFound},
		{"prefix-miss-404", "issues/10", "", domain.ErrNotFound},
		{"literal-pattern-404", "pulls/*", "", domain.ErrNotFound},
		{"traversal-404", "pulls/..", "", domain.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			link, err := svc.RedirectLink(tt.path)
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if link.URL != tt.wantURL {
				t.Errorf("url = %q, want %q", link.URL, tt.wantURL)
			}
		})
	}
}

// TestRedirectLink_ExactWinsOverWildcard seeds the repo directly with a
// multi-segment exact link alongside a matching wildcard; the exact link must
// win. (Exact multi-segment links are not creatable via the API today, but the
// resolver contract still guarantees exact-first.)
func TestRedirectLink_ExactWinsOverWildcard(t *testing.T) {
	repo := newMockRepo()
	svc := app.NewLinkService(repo)
	if _, err := svc.CreateLink("pulls/*", "https://github.com/gjcourt/homelab/pull/*", "", "alice"); err != nil {
		t.Fatalf("create wildcard: %v", err)
	}
	// Seed an exact link at a path the wildcard would also match.
	_ = repo.CreateLink(&domain.Link{Shortcode: "pulls/10", URL: "https://example.com/exact"})

	link, err := svc.RedirectLink("pulls/10")
	if err != nil {
		t.Fatalf("redirect: %v", err)
	}
	if link.URL != "https://example.com/exact" {
		t.Errorf("url = %q, want the exact link %q", link.URL, "https://example.com/exact")
	}
}

// TestRedirectLink_WildcardClickCountsAgainstPattern verifies wildcard hits
// increment the pattern link's counter.
func TestRedirectLink_WildcardClickCountsAgainstPattern(t *testing.T) {
	repo := newMockRepo()
	svc := app.NewLinkService(repo)
	if _, err := svc.CreateLink("pulls/*", "https://example.com/pull/*", "", "alice"); err != nil {
		t.Fatalf("create: %v", err)
	}
	for _, p := range []string{"pulls/1", "pulls/2", "pulls/3"} {
		if _, err := svc.RedirectLink(p); err != nil {
			t.Fatalf("redirect %s: %v", p, err)
		}
	}
	stats, err := svc.GetLinkStats("pulls/*")
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.ClickCount != 3 {
		t.Errorf("click_count = %d, want 3", stats.ClickCount)
	}
}

// TestRedirectLink_UnsafeCaptureNeverInjected is the security regression: a
// captured segment carrying a full URL, a scheme, or an encoded traversal must
// 404 rather than alter the redirect target.
func TestRedirectLink_UnsafeCaptureNeverInjected(t *testing.T) {
	repo := newMockRepo()
	svc := app.NewLinkService(repo)
	if _, err := svc.CreateLink("go-to/*", "https://safe.example.com/x/*", "", "alice"); err != nil {
		t.Fatalf("create: %v", err)
	}
	// These are single path segments (the router hands the service one
	// segment per "*"); each carries an unsafe payload and must be refused.
	for _, path := range []string{
		"go-to/evil.com", // still a valid single-segment host-looking value: allowed but confined to path
		"go-to/%2e%2e",   // encoded traversal
		"go-to/a:b",      // colon (scheme-like)
		"go-to/..",       // dot-dot traversal
	} {
		link, err := svc.RedirectLink(path)
		if err == nil {
			// "go-to/evil.com" is charset-valid; if it resolves it must stay on
			// the original host with the value confined to the path.
			if !strings.HasPrefix(link.URL, "https://safe.example.com/x/") {
				t.Errorf("path %q escaped origin: url = %q", path, link.URL)
			}
			continue
		}
		if err != domain.ErrNotFound {
			t.Errorf("path %q: err = %v, want ErrNotFound", path, err)
		}
	}
}

// Named parameters resolve end to end, and a more specific pattern wins.
func TestRedirectLink_NamedParams(t *testing.T) {
	repo := newMockRepo()
	svc := app.NewLinkService(repo)
	if _, err := svc.CreateLink("gh/{repo}/{pr}", "https://github.com/intrinsic-org/{repo}/pull/{pr}", "", "alice"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := svc.CreateLink("gh/homelab/{pr}", "https://github.com/gjcourt/homelab/pull/{pr}", "", "alice"); err != nil {
		t.Fatalf("create specific: %v", err)
	}
	tests := []struct {
		path, wantURL string
		wantErr       error
	}{
		{"gh/intrinsic/10", "https://github.com/intrinsic-org/intrinsic/pull/10", nil},
		{"gh/homelab/7", "https://github.com/gjcourt/homelab/pull/7", nil},
		{"gh/intrinsic", "", domain.ErrNotFound},
		{"gh/intrinsic/10/files", "", domain.ErrNotFound},
		{"gh/..%2f/10", "", domain.ErrNotFound},
	}
	for _, tt := range tests {
		link, err := svc.RedirectLink(tt.path)
		if tt.wantErr != nil {
			if err != tt.wantErr {
				t.Errorf("%s: err = %v, want %v", tt.path, err, tt.wantErr)
			}
			continue
		}
		if err != nil || link.URL != tt.wantURL {
			t.Errorf("%s: (%v, %v), want %q", tt.path, link, err, tt.wantURL)
		}
	}
	stats, err := svc.GetLinkStats("gh/{repo}/{pr}")
	if err != nil || stats.ClickCount != 1 {
		t.Errorf("clicks on the generic pattern = %v (%v), want 1", stats, err)
	}
}

// Editing a pattern link's destination goes through pattern validation (it
// used to go through plain URL validation, which can't accept placeholders).
func TestUpdateLink_PatternDestination(t *testing.T) {
	repo := newMockRepo()
	svc := app.NewLinkService(repo)
	if _, err := svc.CreateLink("gh/{repo}/{pr}", "https://github.com/a/{repo}/pull/{pr}", "", "alice"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := svc.CreateLink("pulls/*", "https://example.com/pull/*", "", "alice"); err != nil {
		t.Fatalf("create legacy: %v", err)
	}
	if _, err := svc.CreateLink("docs", "https://example.com/docs", "", "alice"); err != nil {
		t.Fatalf("create plain: %v", err)
	}
	if l, err := svc.UpdateLink("gh/{repo}/{pr}", "https://github.com/b/{repo}/pull/{pr}", "", "alice", false); err != nil || l.URL != "https://github.com/b/{repo}/pull/{pr}" {
		t.Errorf("named update: %v %v", l, err)
	}
	if l, err := svc.UpdateLink("pulls/*", "https://example.com/pr/*", "", "alice", false); err != nil || l.URL != "https://example.com/pr/*" {
		t.Errorf("legacy update: %v %v", l, err)
	}
	for sc, bad := range map[string]string{
		"gh/{repo}/{pr}": "https://github.com/b/{repo}", // drops {pr}
		"pulls/*":        "https://example.com/pr/x",    // drops *
		"docs":           "https://example.com/{x}",     // placeholder on a plain link
	} {
		if _, err := svc.UpdateLink(sc, bad, "", "alice", false); err == nil {
			t.Errorf("update %s -> %s: want error", sc, bad)
		}
	}
	if l, _ := svc.GetLink("gh/{repo}/{pr}"); l.URL != "https://github.com/b/{repo}/pull/{pr}" {
		t.Errorf("a rejected update must not change the link: %q", l.URL)
	}
}

// Plain links may carry braces that aren't placeholders — JSON in a query
// string, as Grafana and Kibana links do — on create and update.
func TestPlainLink_BracesAllowed(t *testing.T) {
	svc := app.NewLinkService(newMockRepo())
	grafana := `https://grafana.example.com/explore?left={"q":1}`
	if _, err := svc.CreateLink("dash", grafana, "", "alice"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := svc.UpdateLink("dash", `https://grafana.example.com/explore?left={"q":2}`, "", "alice", false); err != nil {
		t.Fatalf("update: %v", err)
	}
	if _, err := svc.CreateLink("dash2", "https://example.com/{x}", "", "alice"); err == nil {
		t.Error("a real placeholder on a plain link must still be rejected")
	}
}

// Security regression: master's UpdateLink stored a pattern destination
// through plain URL validation, so a row like pulls/* -> https://*/ could
// exist, and go/pulls/evil.com would redirect to evil.com. Whatever wrote
// the row, it must not redirect off the template's host.
func TestRedirectLink_StoredHostPlaceholderRefused(t *testing.T) {
	repo := newMockRepo()
	svc := app.NewLinkService(repo)
	for sc, dest := range map[string]string{
		"pulls/*":   "https://*/",
		"go-to/{h}": "https://{h}/x",
		"sfx/*":     "https://ex*/",
	} {
		_ = repo.CreateLink(&domain.Link{Shortcode: sc, URL: dest})
	}
	for _, path := range []string{"pulls/evil.com", "go-to/evil.com", "sfx/ample.evil.com"} {
		if link, err := svc.RedirectLink(path); err != domain.ErrNotFound {
			t.Errorf("%s: (%v, %v), want ErrNotFound", path, link, err)
		}
	}
}

// A legacy link whose stored destination has a literal {…} still resolves.
func TestRedirectLink_LegacyLiteralBraces(t *testing.T) {
	repo := newMockRepo()
	svc := app.NewLinkService(repo)
	_ = repo.CreateLink(&domain.Link{Shortcode: "s/*", URL: "https://ex.com/search?t={type}&q=*"})
	link, err := svc.RedirectLink("s/go")
	if err != nil || link.URL != "https://ex.com/search?t={type}&q=go" {
		t.Errorf("(%v, %v)", link, err)
	}
}

// Legacy links may carry JSON in the query: creatable, editable, and they
// resolve with the JSON left alone.
func TestLegacyLink_JSONQuery(t *testing.T) {
	svc := app.NewLinkService(newMockRepo())
	dest := `https://g.example.com/explore?left={"q":"up"}&id=*`
	if _, err := svc.CreateLink("logs/*", dest, "", "alice"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := svc.UpdateLink("logs/*", dest, "new description", "alice", false); err != nil {
		t.Fatalf("description-only edit: %v", err)
	}
	link, err := svc.RedirectLink("logs/api")
	if err != nil || link.URL != `https://g.example.com/explore?left={"q":"up"}&id=api` {
		t.Errorf("(%v, %v)", link, err)
	}
}

// An edit that resends a plain link's destination unchanged succeeds even
// if the destination wouldn't pass today's create rules (e.g. a '*' in a
// search query, accepted by older edits).
func TestUpdateLink_UnchangedURLNotRevalidated(t *testing.T) {
	repo := newMockRepo()
	svc := app.NewLinkService(repo)
	_ = repo.CreateLink(&domain.Link{Shortcode: "g", URL: "https://www.google.com/search?q=foo*", Owner: "alice"})
	if _, err := svc.UpdateLink("g", "https://www.google.com/search?q=foo*", "search", "alice", false); err != nil {
		t.Fatalf("description-only edit: %v", err)
	}
	if _, err := svc.UpdateLink("g", "https://www.google.com/search?q=bar*", "", "alice", false); err == nil {
		t.Error("a changed URL is still validated")
	}
}
