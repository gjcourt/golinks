// Package adapthttp provides request handlers.
package adapthttp

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/george/golinks/internal/domain"
	inbound "github.com/george/golinks/internal/ports/inbound"
	outbound "github.com/george/golinks/internal/ports/outbound"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

// ---------- request/response DTOs (transport concern) ----------

// CreateLinkRequest is the JSON body for POST /api/links.
type CreateLinkRequest struct {
	Shortcode   string `json:"shortcode"`
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// UpdateLinkRequest is the JSON body for PUT /api/links/{shortcode}.
type UpdateLinkRequest struct {
	URL         string `json:"url,omitempty"`
	Description string `json:"description,omitempty"`
}

// LinkResponse is the JSON representation returned to clients.
type LinkResponse struct {
	ID          int64  `json:"id"`
	Shortcode   string `json:"shortcode"`
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
	Owner       string `json:"owner,omitempty"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	ClickCount  int64  `json:"click_count"`
}

// StatsResponse is the JSON representation for link statistics.
type StatsResponse struct {
	Shortcode   string `json:"shortcode"`
	ClickCount  int64  `json:"click_count"`
	LastClicked string `json:"last_clicked,omitempty"`
}

func linkToResponse(l *domain.Link) LinkResponse {
	return LinkResponse{
		ID:          l.ID,
		Shortcode:   l.Shortcode,
		URL:         l.URL,
		Description: l.Description,
		Owner:       l.Owner,
		CreatedAt:   l.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   l.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		ClickCount:  l.ClickCount,
	}
}

func statsToResponse(s *domain.LinkStats) StatsResponse {
	r := StatsResponse{
		Shortcode:  s.Shortcode,
		ClickCount: s.ClickCount,
	}
	if !s.LastClicked.IsZero() {
		r.LastClicked = s.LastClicked.Format("2006-01-02T15:04:05Z07:00")
	}
	return r
}

// ---------- Handler ----------

// Handler is the driving HTTP adapter. It depends on inbound.LinkService and outbound.UserRepository.
type Handler struct {
	svc   inbound.LinkService
	auth  AuthConfig
	users outbound.UserRepository
}

// NewHandler creates a Handler wired to the given service, auth config,
// and user repository.
func NewHandler(svc inbound.LinkService, auth AuthConfig, users outbound.UserRepository) *Handler {
	return &Handler{svc: svc, auth: auth, users: users}
}

// RegisterRoutes mounts all routes onto the given router.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	authMW := RequireAuth(h.auth)

	// Protected API routes
	api := r.PathPrefix("/api").Subrouter()
	api.Use(authMW)
	api.HandleFunc("/links", h.ListLinks).Methods("GET")
	api.HandleFunc("/links", h.CreateLink).Methods("POST")
	// {shortcode:.+} allows slashes so wildcard/parameterized links (e.g.
	// "pulls/*") can be managed by their full pattern. The /stats route is
	// registered before the bare-item routes so the greedy ".+" does not
	// swallow the "/stats" suffix.
	api.HandleFunc("/links/{shortcode:.+}/stats", h.GetLinkStats).Methods("GET")
	api.HandleFunc("/links/{shortcode:.+}", h.GetLink).Methods("GET")
	api.HandleFunc("/links/{shortcode:.+}", h.UpdateLink).Methods("PUT")
	api.HandleFunc("/links/{shortcode:.+}", h.DeleteLink).Methods("DELETE")
	api.HandleFunc("/me", h.GetMe).Methods("GET")

	// Login / logout / register — always accessible
	r.HandleFunc("/login", h.LoginPage).Methods("GET")
	r.HandleFunc("/login", h.HandleLogin).Methods("POST")
	r.HandleFunc("/logout", h.HandleLogout).Methods("POST")
	r.HandleFunc("/register", h.RegisterPage).Methods("GET")
	r.HandleFunc("/register", h.HandleRegister).Methods("POST")

	// Protected admin page
	adminHandler := authMW(http.HandlerFunc(h.AdminPage))
	r.Handle("/admin", adminHandler).Methods("GET")
	r.Handle("/admin/", adminHandler).Methods("GET")

	// Public routes. {shortcode:.+} captures multi-segment paths so wildcard
	// links like go/pulls/10 (path "pulls/10") reach the redirect handler;
	// exact and wildcard resolution is decided in the service layer.
	r.HandleFunc("/{shortcode:.+}", h.Redirect).Methods("GET")
	r.HandleFunc("/", h.HomePage).Methods("GET")
}

// ---------- helpers ----------

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// ---------- endpoints ----------

// Redirect handles GET /{shortcode}.
func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	shortcode := mux.Vars(r)["shortcode"]
	link, err := h.svc.RedirectLink(shortcode)
	if err == domain.ErrNotFound {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		slog.Error("redirect failed", "shortcode", shortcode, "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, link.URL, http.StatusFound)
}

// ListLinks handles GET /api/links.
func (h *Handler) ListLinks(w http.ResponseWriter, r *http.Request) {
	links, err := h.svc.ListLinks()
	if err != nil {
		slog.Error("list links failed", "err", err)
		respondError(w, http.StatusInternalServerError, "Failed to list links")
		return
	}
	out := make([]LinkResponse, 0, len(links))
	for _, l := range links {
		out = append(out, linkToResponse(l))
	}
	respondJSON(w, http.StatusOK, out)
}

// CreateLink handles POST /api/links.
func (h *Handler) CreateLink(w http.ResponseWriter, r *http.Request) {
	var req CreateLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	owner := UserFromContext(r.Context())
	link, err := h.svc.CreateLink(req.Shortcode, req.URL, req.Description, owner)
	if err == domain.ErrAlreadyExists {
		respondError(w, http.StatusConflict, "Shortcode already exists")
		return
	}
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, linkToResponse(link))
}

// GetLink handles GET /api/links/{shortcode}.
func (h *Handler) GetLink(w http.ResponseWriter, r *http.Request) {
	shortcode := mux.Vars(r)["shortcode"]
	link, err := h.svc.GetLink(shortcode)
	if err == domain.ErrNotFound {
		respondError(w, http.StatusNotFound, "Link not found")
		return
	}
	if err != nil {
		slog.Error("get link failed", "shortcode", shortcode, "err", err)
		respondError(w, http.StatusInternalServerError, "Failed to get link")
		return
	}
	respondJSON(w, http.StatusOK, linkToResponse(link))
}

// UpdateLink handles PUT /api/links/{shortcode}.
func (h *Handler) UpdateLink(w http.ResponseWriter, r *http.Request) {
	shortcode := mux.Vars(r)["shortcode"]
	var req UpdateLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	username := UserFromContext(r.Context())
	isAdmin := h.isAdmin(username)
	link, err := h.svc.UpdateLink(shortcode, req.URL, req.Description, username, isAdmin)
	if err == domain.ErrNotFound {
		respondError(w, http.StatusNotFound, "Link not found")
		return
	}
	if errors.Is(err, domain.ErrForbidden) {
		respondError(w, http.StatusForbidden, "You are not allowed to update this link")
		return
	}
	if err != nil {
		slog.Error("update link failed", "shortcode", shortcode, "err", err)
		respondError(w, http.StatusInternalServerError, "Failed to update link")
		return
	}
	respondJSON(w, http.StatusOK, linkToResponse(link))
}

// DeleteLink handles DELETE /api/links/{shortcode}.
func (h *Handler) DeleteLink(w http.ResponseWriter, r *http.Request) {
	shortcode := mux.Vars(r)["shortcode"]
	username := UserFromContext(r.Context())
	isAdmin := h.isAdmin(username)
	err := h.svc.DeleteLink(shortcode, username, isAdmin)
	if err == domain.ErrNotFound {
		respondError(w, http.StatusNotFound, "Link not found")
		return
	}
	if errors.Is(err, domain.ErrForbidden) {
		respondError(w, http.StatusForbidden, "You are not allowed to delete this link")
		return
	}
	if err != nil {
		slog.Error("delete link failed", "shortcode", shortcode, "err", err)
		respondError(w, http.StatusInternalServerError, "Failed to delete link")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetLinkStats handles GET /api/links/{shortcode}/stats.
func (h *Handler) GetLinkStats(w http.ResponseWriter, r *http.Request) {
	shortcode := mux.Vars(r)["shortcode"]
	stats, err := h.svc.GetLinkStats(shortcode)
	if err == domain.ErrNotFound {
		respondError(w, http.StatusNotFound, "Link not found")
		return
	}
	if err != nil {
		slog.Error("get stats failed", "shortcode", shortcode, "err", err)
		respondError(w, http.StatusInternalServerError, "Failed to get stats")
		return
	}
	respondJSON(w, http.StatusOK, statsToResponse(stats))
}

// GetMe handles GET /api/me.
func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	username := UserFromContext(r.Context())
	isAdmin := h.isAdmin(username)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"username": username,
		"is_admin": isAdmin,
	})
}

// HomePage serves GET /.
func (h *Handler) HomePage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(homeTemplate))
}

// AdminPage serves GET /admin.
func (h *Handler) AdminPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(adminTemplate))
}

// LoginPage serves GET /login.
func (h *Handler) LoginPage(w http.ResponseWriter, r *http.Request) {
	// If auth is disabled, redirect straight to admin.
	if h.auth.Mode == AuthModeNone {
		http.Redirect(w, r, "/admin", http.StatusFound)
		return
	}
	// If already authenticated via session/proxy, redirect to admin.
	if h.auth.Mode == AuthModeProxy {
		http.Redirect(w, r, "/admin", http.StatusFound)
		return
	}

	// If no users exist, redirect to register
	n, err := h.users.CountUsers()
	if err == nil && n == 0 {
		http.Redirect(w, r, "/register", http.StatusFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(loginTemplate))
}

// HandleLogin handles POST /login (local auth mode only).
func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if h.auth.Mode != AuthModeLocal {
		http.Redirect(w, r, "/admin", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	user, err := h.users.GetUserByUsername(username)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(loginErrorTemplate))
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(loginErrorTemplate))
		return
	}

	// Authentication successful — set session cookie.
	expiry := time.Now().Add(time.Duration(h.auth.CookieMaxAge) * time.Second)
	value := signSession(username, h.auth.Secret, expiry)
	http.SetCookie(w, &http.Cookie{
		Name:     h.auth.CookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   h.auth.CookieMaxAge,
		HttpOnly: true,
		Secure:   h.auth.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/admin", http.StatusFound)
}

// HandleLogout handles POST /logout.
func (h *Handler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.auth.CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

// RegisterPage serves GET /register.
//
// Registration is only exposed when local auth is enabled AND no users yet
// exist (the bootstrap "create the first admin" flow). Once at least one
// user is registered, this page redirects to /login — additional users must
// be created through an authenticated admin flow, not by anonymous visitors.
func (h *Handler) RegisterPage(w http.ResponseWriter, r *http.Request) {
	if h.auth.Mode != AuthModeLocal {
		http.Redirect(w, r, "/admin", http.StatusFound)
		return
	}
	n, err := h.users.CountUsers()
	if err != nil {
		slog.Error("count users failed", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if n > 0 {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(registerTemplate))
}

// HandleRegister handles POST /register — creates the first admin user.
//
// SECURITY: this endpoint is only valid when no users exist yet. After the
// first admin is created, anonymous registration is closed; subsequent
// users must be provisioned by an authenticated admin (a future flow).
// Without this gate, any visitor on a `local`-mode instance could create
// themselves a non-admin account and obtain API access.
func (h *Handler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if h.auth.Mode != AuthModeLocal {
		http.Redirect(w, r, "/admin", http.StatusFound)
		return
	}
	n, err := h.users.CountUsers()
	if err != nil {
		slog.Error("count users failed", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if n > 0 {
		// First admin is already provisioned; refuse silently. Returning
		// 403 rather than redirecting prevents form replays from
		// accidentally creating users via CSRF.
		http.Error(w, "Registration is closed", http.StatusForbidden)
		return
	}

	if parseErr := r.ParseForm(); parseErr != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	username := r.FormValue("username")
	password := r.FormValue("password")
	if username == "" || password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("hash password failed", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	user := &domain.User{
		Username:     username,
		PasswordHash: string(hashed),
		Role:         "admin",
	}
	if err := h.users.CreateUser(user); err != nil {
		slog.Error("create user failed", "username", username, "err", err)
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	slog.Info("user created", "username", username, "role", "admin")
	http.Redirect(w, r, "/login", http.StatusFound)
}

// isAdmin checks whether the given username has the admin role.
func (h *Handler) isAdmin(username string) bool {
	// In none mode authentication is disabled and there is never a username,
	// yet everyone is effectively an admin. This check must come before the
	// empty-username guard below — otherwise none-mode requests (which always
	// carry an empty username) would be reported as non-admin, hiding the
	// Edit/Delete controls on the admin page and causing the API to reject
	// updates to any link with a non-empty owner as forbidden.
	if h.auth.Mode == AuthModeNone {
		return true
	}
	if username == "" {
		return false
	}
	// In local/proxy mode, admin is determined by the user's stored role.
	u, err := h.users.GetUserByUsername(username)
	if err != nil {
		return false
	}
	return u.Role == "admin"
}
