package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/george/golinks/internal/domain"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

// ---------------------------------------------------------------------------
// Mock LinkService
// ---------------------------------------------------------------------------

type mockService struct {
	links map[string]*domain.Link
}

func newMockService() *mockService {
	return &mockService{links: make(map[string]*domain.Link)}
}

func (m *mockService) CreateLink(shortcode, url, desc string) (*domain.Link, error) {
	if _, ok := m.links[shortcode]; ok {
		return nil, domain.ErrAlreadyExists
	}
	l := &domain.Link{
		ID: int64(len(m.links) + 1), Shortcode: shortcode, URL: url,
		Description: desc, CreatedAt: time.Now(), UpdatedAt: time.Now(),
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

func (m *mockService) UpdateLink(shortcode, url, desc string) (*domain.Link, error) {
	l, ok := m.links[shortcode]
	if !ok {
		return nil, domain.ErrNotFound
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

func (m *mockService) DeleteLink(shortcode string) error {
	if _, ok := m.links[shortcode]; !ok {
		return domain.ErrNotFound
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

// helpers

func setupRouter(svc domain.LinkService) *mux.Router {
	r := mux.NewRouter()
	NewHandler(svc, DefaultAuthConfig()).RegisterRoutes(r)
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
	json.NewDecoder(w.Body).Decode(&links)
	if len(links) != 0 {
		t.Errorf("expected 0 links, got %d", len(links))
	}
}

func TestUpdateLink_OK(t *testing.T) {
	svc := newMockService()
	seedLink(svc, "docs", "https://old.com")
	r := setupRouter(svc)
	body, _ := json.Marshal(UpdateLinkRequest{URL: "https://new.com"})
	req := httptest.NewRequest("PUT", "/api/links/docs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var resp LinkResponse
	json.NewDecoder(w.Body).Decode(&resp)
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

func TestRedirect_NotFound(t *testing.T) {
	r := setupRouter(newMockService())
	req := httptest.NewRequest("GET", "/nope", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
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
	NewHandler(newMockService(), cfg).RegisterRoutes(r)

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
	cfg.Username = "admin"
	// bcrypt hash of "password123"
	cfg.HashedPassword, _ = bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	cfg.Secret = secret
	NewHandler(newMockService(), cfg).RegisterRoutes(r)

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
	cfg.Username = "admin"
	cfg.HashedPassword, _ = bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.DefaultCost)
	cfg.Secret = secret
	NewHandler(newMockService(), cfg).RegisterRoutes(r)

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
	cfg.Username = "admin"
	cfg.HashedPassword, _ = bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	cfg.Secret = secret
	NewHandler(newMockService(), cfg).RegisterRoutes(r)

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
	cfg.Username = "admin"
	cfg.HashedPassword, _ = bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	cfg.Secret = secret
	NewHandler(newMockService(), cfg).RegisterRoutes(r)

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
	cfg.Username = "admin"
	cfg.HashedPassword, _ = bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	cfg.Secret = secret
	cfg.APIKey = "my-secret-api-key"
	NewHandler(newMockService(), cfg).RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/api/links", nil)
	req.Header.Set("Authorization", "Bearer my-secret-api-key")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}
