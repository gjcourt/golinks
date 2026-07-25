package adapthttp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/george/golinks/internal/app"
	"github.com/george/golinks/internal/domain"
	inbound "github.com/george/golinks/internal/ports/inbound"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

// ---------------------------------------------------------------------------
// Mock LinkService
// ---------------------------------------------------------------------------

const testUser = "admin"

type mockService struct {
	links map[string]*domain.Link
}

func newMockService() *mockService {
	return &mockService{links: make(map[string]*domain.Link)}
}

func (m *mockService) CreateLink(shortcode, url, desc, owner string) (*domain.Link, error) {
	if _, ok := m.links[shortcode]; ok {
		return nil, domain.ErrAlreadyExists
	}
	l := &domain.Link{
		ID: int64(len(m.links) + 1), Shortcode: shortcode, URL: url,
		Description: desc, Owner: owner, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	m.links[shortcode] = l
	return l, nil
}

func (m *mockService) GetLink(shortcode string) (*domain.Link, error) {
	l, ok := m.links[shortcode]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return l, nil
}

func (m *mockService) UpdateLink(shortcode, url, desc, username string, isAdmin bool) (*domain.Link, error) {
	l, ok := m.links[shortcode]
	if !ok {
		return nil, domain.ErrNotFound
	}
	if !isAdmin && l.Owner != username {
		return nil, domain.ErrForbidden
	}
	if url != "" {
		l.URL = url
	}
	if desc != "" {
		l.Description = desc
	}
	l.UpdatedAt = time.Now()
	return l, nil
}

func (m *mockService) DeleteLink(shortcode, username string, isAdmin bool) error {
	l, ok := m.links[shortcode]
	if !ok {
		return domain.ErrNotFound
	}
	if !isAdmin && l.Owner != username {
		return domain.ErrForbidden
	}
	delete(m.links, shortcode)
	return nil
}

func (m *mockService) ListLinks() ([]*domain.Link, error) {
	out := make([]*domain.Link, 0, len(m.links))
	for _, l := range m.links {
		out = append(out, l)
	}
	return out, nil
}

func (m *mockService) RedirectLink(shortcode string) (*domain.Link, error) {
	l, ok := m.links[shortcode]
	if !ok {
		return nil, domain.ErrNotFound
	}
	l.ClickCount++
	return l, nil
}

func (m *mockService) GetLinkStats(shortcode string) (*domain.LinkStats, error) {
	l, ok := m.links[shortcode]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &domain.LinkStats{Shortcode: shortcode, ClickCount: l.ClickCount}, nil
}

// ---------------------------------------------------------------------------
// Mock UserRepository
// ---------------------------------------------------------------------------

type mockUserRepo struct {
	users map[string]*domain.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*domain.User)}
}

func (m *mockUserRepo) CreateUser(user *domain.User) error {
	if _, ok := m.users[user.Username]; ok {
		return domain.ErrAlreadyExists
	}
	user.ID = int64(len(m.users) + 1)
	user.CreatedAt = time.Now()
	stored := *user
	m.users[user.Username] = &stored
	return nil
}

func (m *mockUserRepo) GetUserByUsername(username string) (*domain.User, error) {
	u, ok := m.users[username]
	if !ok {
		return nil, domain.ErrNotFound
	}
	copy := *u
	return &copy, nil
}

func (m *mockUserRepo) CountUsers() (int64, error) {
	return int64(len(m.users)), nil
}

func (m *mockUserRepo) Close() error { return nil }

// seedUser adds a user to the mock repo with the given plaintext password and role.
func seedUser(repo *mockUserRepo, username, password, role string) {
	h, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	repo.users[username] = &domain.User{
		ID:           int64(len(repo.users) + 1),
		Username:     username,
		PasswordHash: string(h),
		Role:         role,
		CreatedAt:    time.Now(),
	}
}

// helpers

func setupRouter(svc inbound.LinkService) *mux.Router {
	r := mux.NewRouter()
	NewHandler(svc, DefaultAuthConfig(), newMockUserRepo()).RegisterRoutes(r)
	return r
}

func seedLink(svc *mockService, shortcode, url string) {
	svc.links[shortcode] = &domain.Link{
		ID: 1, Shortcode: shortcode, URL: url,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestCreateLink_Created(t *testing.T) {
	svc := newMockService()
	r := setupRouter(svc)
	body, _ := json.Marshal(CreateLinkRequest{Shortcode: "docs", URL: "https://docs.example.com"})
	req := httptest.NewRequest("POST", "/api/links", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestCreateLink_Conflict(t *testing.T) {
	svc := newMockService()
	seedLink(svc, "docs", "https://a.com")
	r := setupRouter(svc)
	body, _ := json.Marshal(CreateLinkRequest{Shortcode: "docs", URL: "https://b.com"})
	req := httptest.NewRequest("POST", "/api/links", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestGetLink_OK(t *testing.T) {
	svc := newMockService()
	seedLink(svc, "docs", "https://docs.example.com")
	r := setupRouter(svc)
	req := httptest.NewRequest("GET", "/api/links/docs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestGetLink_NotFound(t *testing.T) {
	r := setupRouter(newMockService())
	req := httptest.NewRequest("GET", "/api/links/nope", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestListLinks_Empty(t *testing.T) {
	r := setupRouter(newMockService())
	req := httptest.NewRequest("GET", "/api/links", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var links []LinkResponse
	_ = json.NewDecoder(w.Body).Decode(&links)
	if len(links) != 0 {
		t.Errorf("expected 0 links, got %d", len(links))
	}
}

func TestUpdateLink_OK(t *testing.T) {
	svc := newMockService()
	seedLink(svc, "olddocs", "https://old.com")
	r := setupRouter(svc)
	body, _ := json.Marshal(UpdateLinkRequest{URL: "https://new.com"})
	req := httptest.NewRequest("PUT", "/api/links/olddocs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var resp LinkResponse
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp.URL != "https://new.com" {
		t.Errorf("url = %q, want %q", resp.URL, "https://new.com")
	}
}

func TestUpdateLink_NotFound(t *testing.T) {
	r := setupRouter(newMockService())
	body, _ := json.Marshal(UpdateLinkRequest{URL: "https://x.com"})
	req := httptest.NewRequest("PUT", "/api/links/nope", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestDeleteLink_NoContent(t *testing.T) {
	svc := newMockService()
	seedLink(svc, "docs", "https://a.com")
	r := setupRouter(svc)
	req := httptest.NewRequest("DELETE", "/api/links/docs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestDeleteLink_NotFound(t *testing.T) {
	r := setupRouter(newMockService())
	req := httptest.NewRequest("DELETE", "/api/links/nope", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// TestGetMe_NoneMode_IsAdmin verifies that when auth is disabled (the default
// deployment mode) /api/me reports the caller as an admin. The admin page hides
// the per-row Edit/Delete controls unless is_admin is true (or the caller owns
// the link), so a false value here leaves every row's Actions column empty.
func TestGetMe_NoneMode_IsAdmin(t *testing.T) {
	r := setupRouter(newMockService()) // DefaultAuthConfig = none
	req := httptest.NewRequest("GET", "/api/me", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var me struct {
		Username string `json:"username"`
		IsAdmin  bool   `json:"is_admin"`
	}
	if err := json.NewDecoder(w.Body).Decode(&me); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !me.IsAdmin {
		t.Errorf("is_admin = false in none mode; want true (Edit/Delete controls would be hidden)")
	}
}

// TestUpdateLink_NoneMode_ForeignOwner verifies that in none mode a link owned
// by someone else can still be edited — everyone is effectively an admin, so the
// ownership check must not forbid the update.
func TestUpdateLink_NoneMode_ForeignOwner(t *testing.T) {
	svc := newMockService()
	svc.links["home"] = &domain.Link{
		ID: 1, Shortcode: "home", URL: "https://old.example.com",
		Owner: "someoneelse", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	r := setupRouter(svc)
	body, _ := json.Marshal(UpdateLinkRequest{URL: "https://new.example.com"})
	req := httptest.NewRequest("PUT", "/api/links/home", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (none-mode admin should edit any link)", w.Code, http.StatusOK)
	}
	var resp LinkResponse
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp.URL != "https://new.example.com" {
		t.Errorf("url = %q, want %q (update did not take effect)", resp.URL, "https://new.example.com")
	}
}

// TestDeleteLink_NoneMode_ForeignOwner verifies that in none mode a link owned
// by someone else can still be deleted.
func TestDeleteLink_NoneMode_ForeignOwner(t *testing.T) {
	svc := newMockService()
	svc.links["home"] = &domain.Link{
		ID: 1, Shortcode: "home", URL: "https://old.example.com",
		Owner: "someoneelse", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	r := setupRouter(svc)
	req := httptest.NewRequest("DELETE", "/api/links/home", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d (none-mode admin should delete any link)", w.Code, http.StatusNoContent)
	}
}

// TestIsAdmin_ModeOrdering pins the branch ordering in isAdmin. Moving the
// AuthModeNone short-circuit above the empty-username guard fixes none mode, but
// it must NOT leak admin into local/proxy mode: those modes still fall through to
// the empty-username guard (false) and the role lookup. An empty username in
// local/proxy must never be admin, and a non-admin user must never be admin.
func TestIsAdmin_ModeOrdering(t *testing.T) {
	tests := []struct {
		name     string
		mode     AuthMode
		username string
		role     string // role to seed for username (empty = don't seed)
		want     bool
	}{
		{"none empty username is admin", AuthModeNone, "", "", true},
		{"none arbitrary username is admin", AuthModeNone, "nobody", "", true},
		{"local empty username not admin", AuthModeLocal, "", "", false},
		{"local non-admin user not admin", AuthModeLocal, "alice", "user", false},
		{"local admin user is admin", AuthModeLocal, "admin", "admin", true},
		{"proxy empty username not admin", AuthModeProxy, "", "", false},
		{"proxy non-admin user not admin", AuthModeProxy, "bob", "user", false},
		{"proxy admin user is admin", AuthModeProxy, "root", "admin", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultAuthConfig()
			cfg.Mode = tt.mode
			userRepo := newMockUserRepo()
			if tt.role != "" {
				_ = userRepo.CreateUser(&domain.User{Username: tt.username, Role: tt.role})
			}
			h := NewHandler(newMockService(), cfg, userRepo)
			if got := h.isAdmin(tt.username); got != tt.want {
				t.Errorf("isAdmin(%q) in %s mode = %v, want %v", tt.username, tt.mode, got, tt.want)
			}
		})
	}
}

func TestRedirect_Found(t *testing.T) {
	svc := newMockService()
	seedLink(svc, "docs", "https://docs.example.com")
	r := setupRouter(svc)
	req := httptest.NewRequest("GET", "/docs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != "https://docs.example.com" {
		t.Errorf("Location = %q, want %q", loc, "https://docs.example.com")
	}
}

// TestRedirect_NotFound verifies that an unknown shortcode no longer 404s but
// instead 302-redirects to the admin create-link page with the missing
// shortcode pre-filled via the ?new= query param.
func TestRedirect_NotFound(t *testing.T) {
	r := setupRouter(newMockService())
	req := httptest.NewRequest("GET", "/nope", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != "/admin?new=nope" {
		t.Errorf("Location = %q, want %q", loc, "/admin?new=nope")
	}
}

// TestRedirect_NotFound_WildcardShortcodeEscaped verifies a multi-segment /
// slash-bearing missing shortcode is query-escaped in the redirect target.
func TestRedirect_NotFound_WildcardShortcodeEscaped(t *testing.T) {
	r := setupRouter(newMockService())
	req := httptest.NewRequest("GET", "/pulls/123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != "/admin?new=pulls%2F123" {
		t.Errorf("Location = %q, want %q", loc, "/admin?new=pulls%2F123")
	}
}

// ---------------------------------------------------------------------------
// Wildcard redirect — end-to-end through the real service and router
// ---------------------------------------------------------------------------

// fakeRepo is a minimal in-file LinkRepository so these tests can drive the
// real app.linkService (and thus real wildcard resolution) without importing a
// sibling storage adapter (forbidden by depguard).
type fakeRepo struct {
	links map[string]*domain.Link
}

func newFakeRepo() *fakeRepo { return &fakeRepo{links: make(map[string]*domain.Link)} }

func (f *fakeRepo) CreateLink(l *domain.Link) error {
	if _, ok := f.links[l.Shortcode]; ok {
		return domain.ErrAlreadyExists
	}
	cp := *l
	cp.ID = int64(len(f.links) + 1)
	f.links[l.Shortcode] = &cp
	return nil
}
func (f *fakeRepo) GetLink(sc string) (*domain.Link, error) {
	l, ok := f.links[sc]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *l
	return &cp, nil
}
func (f *fakeRepo) UpdateLink(sc string, l *domain.Link) error {
	if _, ok := f.links[sc]; !ok {
		return domain.ErrNotFound
	}
	cp := *l
	f.links[sc] = &cp
	return nil
}
func (f *fakeRepo) DeleteLink(sc string) error {
	if _, ok := f.links[sc]; !ok {
		return domain.ErrNotFound
	}
	delete(f.links, sc)
	return nil
}
func (f *fakeRepo) ListLinks() ([]*domain.Link, error) {
	out := make([]*domain.Link, 0, len(f.links))
	for _, l := range f.links {
		cp := *l
		out = append(out, &cp)
	}
	return out, nil
}
func (f *fakeRepo) IncrementClickCount(sc string) error {
	if l, ok := f.links[sc]; ok {
		l.ClickCount++
	}
	return nil
}
func (f *fakeRepo) GetStats(sc string) (*domain.LinkStats, error) {
	l, ok := f.links[sc]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &domain.LinkStats{Shortcode: sc, ClickCount: l.ClickCount}, nil
}
func (f *fakeRepo) Close() error { return nil }

func TestRedirect_Wildcard_EndToEnd(t *testing.T) {
	repo := newFakeRepo()
	svc := app.NewLinkService(repo)
	if _, err := svc.CreateLink("pulls/*", "https://github.com/gjcourt/homelab/pull/*", "", "admin"); err != nil {
		t.Fatalf("create: %v", err)
	}
	r := mux.NewRouter()
	NewHandler(svc, DefaultAuthConfig(), newMockUserRepo()).RegisterRoutes(r)

	tests := []struct {
		name, path string
		wantCode   int
		wantLoc    string
	}{
		{"resolves", "/pulls/10", http.StatusFound, "https://github.com/gjcourt/homelab/pull/10"},
		// Non-resolving shortcodes no longer 404: they 302 to the admin
		// create-link page with the (query-escaped) shortcode pre-filled.
		{"multi-segment-create", "/pulls/10/extra", http.StatusFound, "/admin?new=pulls%2F10%2Fextra"},
		// A colon (scheme-like) survives path cleaning but fails the capture
		// charset check in the resolver -> not resolved, offered for creation.
		{"unsafe-capture-create", "/pulls/a:b", http.StatusFound, "/admin?new=pulls%2Fa%3Ab"},
		{"prefix-miss-create", "/issues/10", http.StatusFound, "/admin?new=issues%2F10"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantCode)
			}
			if tt.wantLoc != "" {
				if loc := w.Header().Get("Location"); loc != tt.wantLoc {
					t.Errorf("Location = %q, want %q", loc, tt.wantLoc)
				}
			}
		})
	}
}

// TestManageWildcardLink_ViaAPI confirms a wildcard link (shortcode with "/"
// and "*") is reachable through the slash-tolerant API item routes for
// stats and delete.
func TestManageWildcardLink_ViaAPI(t *testing.T) {
	repo := newFakeRepo()
	svc := app.NewLinkService(repo)
	if _, err := svc.CreateLink("pulls/*", "https://example.com/pull/*", "", "admin"); err != nil {
		t.Fatalf("create: %v", err)
	}
	r := mux.NewRouter()
	NewHandler(svc, DefaultAuthConfig(), newMockUserRepo()).RegisterRoutes(r)

	// Stats for the pattern shortcode.
	req := httptest.NewRequest("GET", "/api/links/pulls/*/stats", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("stats status = %d, want %d", w.Code, http.StatusOK)
	}

	// Delete the pattern shortcode.
	req = httptest.NewRequest("DELETE", "/api/links/pulls/*", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", w.Code, http.StatusNoContent)
	}
	if _, err := repo.GetLink("pulls/*"); err != domain.ErrNotFound {
		t.Errorf("wildcard link not deleted: %v", err)
	}
}

func TestGetLinkStats_OK(t *testing.T) {
	svc := newMockService()
	seedLink(svc, "docs", "https://docs.example.com")
	r := setupRouter(svc)
	req := httptest.NewRequest("GET", "/api/links/docs/stats", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestGetLinkStats_NotFound(t *testing.T) {
	r := setupRouter(newMockService())
	req := httptest.NewRequest("GET", "/api/links/nope/stats", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHomePage(t *testing.T) {
	r := setupRouter(newMockService())
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/html; charset=utf-8", ct)
	}
}

func TestAdminPage(t *testing.T) {
	r := setupRouter(newMockService())
	req := httptest.NewRequest("GET", "/admin", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestLoginPage(t *testing.T) {
	r := mux.NewRouter()
	cfg := DefaultAuthConfig()
	cfg.Mode = AuthModeLocal

	userRepo := newMockUserRepo()
	_ = userRepo.CreateUser(&domain.User{Username: "admin", Role: "admin"})

	NewHandler(newMockService(), cfg, userRepo).RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/html; charset=utf-8", ct)
	}
}

func TestLoginPage_NoneMode_RedirectsToAdmin(t *testing.T) {
	r := setupRouter(newMockService()) // default = none
	req := httptest.NewRequest("GET", "/login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != "/admin" {
		t.Errorf("Location = %q, want /admin", loc)
	}
}

func TestHandleLogin_Success(t *testing.T) {
	r := mux.NewRouter()
	secret, _ := GenerateRandomSecret()
	cfg := DefaultAuthConfig()
	cfg.Mode = AuthModeLocal
	cfg.Secret = secret

	userRepo := newMockUserRepo()
	seedUser(userRepo, testUser, "password123", "admin")
	NewHandler(newMockService(), cfg, userRepo).RegisterRoutes(r)

	body := "username=admin&password=password123"
	req := httptest.NewRequest("POST", "/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != "/admin" {
		t.Errorf("Location = %q, want /admin", loc)
	}
	// Should have set a session cookie
	cookies := w.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == cfg.CookieName {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected session cookie to be set")
	}
}

func TestHandleLogin_WrongPassword(t *testing.T) {
	r := mux.NewRouter()
	secret, _ := GenerateRandomSecret()
	cfg := DefaultAuthConfig()
	cfg.Mode = AuthModeLocal
	cfg.Secret = secret

	userRepo := newMockUserRepo()
	seedUser(userRepo, testUser, "correct", "admin")
	NewHandler(newMockService(), cfg, userRepo).RegisterRoutes(r)

	body := "username=admin&password=wrong"
	req := httptest.NewRequest("POST", "/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleLogout(t *testing.T) {
	r := setupRouter(newMockService())
	req := httptest.NewRequest("POST", "/logout", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != "/" {
		t.Errorf("Location = %q, want /", loc)
	}
	// Cookie should be cleared
	for _, c := range w.Result().Cookies() {
		if c.Name == "golinks_session" && c.MaxAge != -1 {
			t.Error("expected cookie MaxAge=-1 to clear it")
		}
	}
}

func TestAdminPage_AuthRequired(t *testing.T) {
	r := mux.NewRouter()
	secret, _ := GenerateRandomSecret()
	cfg := DefaultAuthConfig()
	cfg.Mode = AuthModeLocal
	cfg.Secret = secret

	userRepo := newMockUserRepo()
	seedUser(userRepo, testUser, "pass", "admin")
	NewHandler(newMockService(), cfg, userRepo).RegisterRoutes(r)

	// Without cookie — browser request should redirect to /login
	req := httptest.NewRequest("GET", "/admin", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != "/login" {
		t.Errorf("Location = %q, want /login", loc)
	}
}

func TestAPI_AuthRequired(t *testing.T) {
	r := mux.NewRouter()
	secret, _ := GenerateRandomSecret()
	cfg := DefaultAuthConfig()
	cfg.Mode = AuthModeLocal
	cfg.Secret = secret

	userRepo := newMockUserRepo()
	seedUser(userRepo, "regularuser", "pass", "user")
	NewHandler(newMockService(), cfg, userRepo).RegisterRoutes(r)

	// API request without auth should get 401
	req := httptest.NewRequest("GET", "/api/links", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAPI_WithAPIKey(t *testing.T) {
	r := mux.NewRouter()
	secret, _ := GenerateRandomSecret()
	cfg := DefaultAuthConfig()
	cfg.Mode = AuthModeLocal
	cfg.Secret = secret
	cfg.APIKey = "my-secret-api-key"

	userRepo := newMockUserRepo()
	seedUser(userRepo, testUser, "pass", "admin")
	NewHandler(newMockService(), cfg, userRepo).RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/api/links", nil)
	req.Header.Set("Authorization", "Bearer my-secret-api-key")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}
