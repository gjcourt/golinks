// Package http provides request handlers.
package http

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/george/golinks/internal/domain"
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

// Handler is the driving HTTP adapter. It depends only on domain.LinkService.
type Handler struct {
	svc   domain.LinkService
	auth  AuthConfig
	users domain.UserRepository
}

// NewHandler creates a Handler wired to the given service, auth config,
// and user repository.
func NewHandler(svc domain.LinkService, auth AuthConfig, users domain.UserRepository) *Handler {
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
	api.HandleFunc("/links/{shortcode}", h.GetLink).Methods("GET")
	api.HandleFunc("/links/{shortcode}", h.UpdateLink).Methods("PUT")
	api.HandleFunc("/links/{shortcode}", h.DeleteLink).Methods("DELETE")
	api.HandleFunc("/links/{shortcode}/stats", h.GetLinkStats).Methods("GET")

	// Login / logout / setup — always accessible
	r.HandleFunc("/login", h.LoginPage).Methods("GET")
	r.HandleFunc("/login", h.HandleLogin).Methods("POST")
	r.HandleFunc("/logout", h.HandleLogout).Methods("POST")
	r.HandleFunc("/setup", h.SetupPage).Methods("GET")
	r.HandleFunc("/setup", h.HandleSetup).Methods("POST")

	// Protected admin page
	adminHandler := authMW(http.HandlerFunc(h.AdminPage))
	r.Handle("/admin", adminHandler).Methods("GET")
	r.Handle("/admin/", adminHandler).Methods("GET")

	// Public routes
	r.HandleFunc("/{shortcode}", h.Redirect).Methods("GET")
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
		log.Printf("error redirecting %q: %v", shortcode, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, link.URL, http.StatusFound)
}

// ListLinks handles GET /api/links.
func (h *Handler) ListLinks(w http.ResponseWriter, r *http.Request) {
	links, err := h.svc.ListLinks()
	if err != nil {
		log.Printf("error listing links: %v", err)
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
		log.Printf("error getting link %q: %v", shortcode, err)
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
		log.Printf("error updating link %q: %v", shortcode, err)
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
		log.Printf("error deleting link %q: %v", shortcode, err)
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
		log.Printf("error getting stats for %q: %v", shortcode, err)
		respondError(w, http.StatusInternalServerError, "Failed to get stats")
		return
	}
	respondJSON(w, http.StatusOK, statsToResponse(stats))
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

// SetupPage serves GET /setup.
// It redirects to /login if at least one user already exists.
func (h *Handler) SetupPage(w http.ResponseWriter, r *http.Request) {
	if h.auth.Mode != AuthModeLocal {
		http.Redirect(w, r, "/admin", http.StatusFound)
		return
	}
	n, err := h.users.CountUsers()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if n > 0 {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(setupTemplate))
}

// HandleSetup handles POST /setup — creates the first admin user.
func (h *Handler) HandleSetup(w http.ResponseWriter, r *http.Request) {
	if h.auth.Mode != AuthModeLocal {
		http.Redirect(w, r, "/admin", http.StatusFound)
		return
	}
	n, err := h.users.CountUsers()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if n > 0 {
		http.Redirect(w, r, "/login", http.StatusFound)
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
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	user := &domain.User{
		Username:     username,
		PasswordHash: string(hashed),
		Role:         "admin",
	}
	if err := h.users.CreateUser(user); err != nil {
		http.Error(w, "Failed to create user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("First admin user %q created via setup", username)
	http.Redirect(w, r, "/login", http.StatusFound)
}

// isAdmin checks whether the given username has the admin role.
func (h *Handler) isAdmin(username string) bool {
	if username == "" {
		return false
	}
	// In proxy mode, the first user or env-configured admin gets admin.
	// In none mode, everyone is effectively admin.
	if h.auth.Mode == AuthModeNone {
		return true
	}
	u, err := h.users.GetUserByUsername(username)
	if err != nil {
		return false
	}
	return u.Role == "admin"
}
