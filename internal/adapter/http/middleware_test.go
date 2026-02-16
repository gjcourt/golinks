package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func protectedHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello " + user))
	})
}

func TestRequireAuth_NoneMode(t *testing.T) {
	cfg := DefaultAuthConfig()
	cfg.Mode = AuthModeNone

	handler := RequireAuth(cfg)(protectedHandler())
	req := httptest.NewRequest("GET", "/api/links", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRequireAuth_LocalMode(t *testing.T) {
	const (
		testUser  = "admin"
		adminPath = "/admin"
	)
	secret, _ := GenerateRandomSecret()
	cfg := DefaultAuthConfig()
	cfg.Mode = AuthModeLocal
	cfg.Username = testUser
	cfg.Secret = secret

	mw := RequireAuth(cfg)

	tests := []struct {
		name       string
		setup      func(r *http.Request)
		wantStatus int
		wantBody   string
		apiPath    bool
	}{
		{
			name:       "no-cookie-api-returns-401",
			setup:      func(r *http.Request) {},
			wantStatus: http.StatusUnauthorized,
			apiPath:    true,
		},
		{
			name:  "no-cookie-browser-redirects-to-login",
			setup: func(r *http.Request) {},
			// /admin is not /api/ so browser redirect
			wantStatus: http.StatusFound,
			apiPath:    false,
		},
		{
			name: "valid-cookie-passes",
			setup: func(r *http.Request) {
				expiry := time.Now().Add(1 * time.Hour)
				val := signSession(testUser, secret, expiry)
				r.AddCookie(&http.Cookie{Name: cfg.CookieName, Value: val})
			},
			wantStatus: http.StatusOK,
			wantBody:   "hello " + testUser,
			apiPath:    true,
		},
		{
			name: "expired-cookie-rejected",
			setup: func(r *http.Request) {
				expiry := time.Now().Add(-1 * time.Hour)
				val := signSession(testUser, secret, expiry)
				r.AddCookie(&http.Cookie{Name: cfg.CookieName, Value: val})
			},
			wantStatus: http.StatusUnauthorized,
			apiPath:    true,
		},
		{
			name: "tampered-cookie-rejected",
			setup: func(r *http.Request) {
				r.AddCookie(&http.Cookie{Name: cfg.CookieName, Value: "tampered.value"})
			},
			wantStatus: http.StatusUnauthorized,
			apiPath:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := adminPath
			if tt.apiPath {
				path = "/api/links"
			}
			req := httptest.NewRequest("GET", path, nil)
			tt.setup(req)
			w := httptest.NewRecorder()
			mw(protectedHandler()).ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.wantBody != "" && w.Body.String() != tt.wantBody {
				t.Errorf("body = %q, want %q", w.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestRequireAuth_APIKey(t *testing.T) {
	cfg := DefaultAuthConfig()
	cfg.Mode = AuthModeLocal
	cfg.APIKey = "test-secret-key-123"
	cfg.Secret = []byte("unused-for-this-test")

	mw := RequireAuth(cfg)

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{"valid-key", "Bearer test-secret-key-123", http.StatusOK},
		{"wrong-key", "Bearer wrong-key", http.StatusUnauthorized},
		{"no-header", "", http.StatusUnauthorized},
		{"bad-prefix", "Token test-secret-key-123", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/links", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()
			mw(protectedHandler()).ServeHTTP(w, req)
			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestRequireAuth_ProxyMode(t *testing.T) {
	cfg := DefaultAuthConfig()
	cfg.Mode = AuthModeProxy
	cfg.ProxyHeader = "Remote-User"

	tests := []struct {
		name           string
		header         string
		trustedProxies []string
		remoteAddr     string
		wantStatus     int
		wantBody       string
	}{
		{
			name:       "header-present-no-trusted-list",
			header:     "alice",
			remoteAddr: "10.0.0.1:12345",
			wantStatus: http.StatusOK,
			wantBody:   "hello alice",
		},
		{
			name:       "no-header",
			header:     "",
			remoteAddr: "10.0.0.1:12345",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:           "trusted-proxy-allowed",
			header:         "bob",
			trustedProxies: []string{"10.0.0.0/8"},
			remoteAddr:     "10.0.0.1:12345",
			wantStatus:     http.StatusOK,
			wantBody:       "hello bob",
		},
		{
			name:           "untrusted-proxy-blocked",
			header:         "bob",
			trustedProxies: []string{"192.168.1.0/24"},
			remoteAddr:     "10.0.0.1:12345",
			wantStatus:     http.StatusUnauthorized,
		},
		{
			name:           "single-ip-trusted",
			header:         "carol",
			trustedProxies: []string{"172.16.0.5"},
			remoteAddr:     "172.16.0.5:9999",
			wantStatus:     http.StatusOK,
			wantBody:       "hello carol",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			localCfg := cfg
			localCfg.TrustedProxies = tt.trustedProxies
			mw := RequireAuth(localCfg)

			req := httptest.NewRequest("GET", "/api/links", nil)
			if tt.header != "" {
				req.Header.Set("Remote-User", tt.header)
			}
			req.RemoteAddr = tt.remoteAddr
			w := httptest.NewRecorder()
			mw(protectedHandler()).ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.wantBody != "" && w.Body.String() != tt.wantBody {
				t.Errorf("body = %q, want %q", w.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestSessionSignVerify(t *testing.T) {
	secret, _ := GenerateRandomSecret()

	t.Run("round-trip", func(t *testing.T) {
		expiry := time.Now().Add(1 * time.Hour)
		token := signSession("testuser", secret, expiry)
		user, ok := verifySession(token, secret)
		if !ok {
			t.Fatal("expected valid session")
		}
		if user != "testuser" {
			t.Errorf("user = %q, want %q", user, "testuser")
		}
	})

	t.Run("wrong-secret", func(t *testing.T) {
		other, _ := GenerateRandomSecret()
		expiry := time.Now().Add(1 * time.Hour)
		token := signSession("testuser", secret, expiry)
		_, ok := verifySession(token, other)
		if ok {
			t.Fatal("expected invalid session with wrong secret")
		}
	})

	t.Run("expired", func(t *testing.T) {
		expiry := time.Now().Add(-1 * time.Minute)
		token := signSession("testuser", secret, expiry)
		_, ok := verifySession(token, secret)
		if ok {
			t.Fatal("expected expired session to be rejected")
		}
	})
}

func TestUserFromContext_Empty(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	if user := UserFromContext(req.Context()); user != "" {
		t.Errorf("expected empty user, got %q", user)
	}
}

func TestParseTrustedProxies(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  int // number of parsed nets
	}{
		{"empty", nil, 0},
		{"single-ip", []string{"10.0.0.1"}, 1},
		{"cidr", []string{"192.168.0.0/16"}, 1},
		{"mixed", []string{"10.0.0.1", "172.16.0.0/12"}, 2},
		{"invalid-skipped", []string{"not-an-ip", "10.0.0.1"}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nets := parseTrustedProxies(tt.input)
			if len(nets) != tt.want {
				t.Errorf("got %d nets, want %d", len(nets), tt.want)
			}
		})
	}
}
