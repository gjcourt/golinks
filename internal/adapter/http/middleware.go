package http

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// ---------- auth mode ----------

// AuthMode defines how authentication is performed.
type AuthMode string

const (
	// AuthModeNone disables authentication.
	AuthModeNone AuthMode = "none"
	// AuthModeLocal uses single-user username/password with session cookie.
	AuthModeLocal AuthMode = "local"
	// AuthModeProxy trusts a reverse-proxy header (e.g. Authelia).
	AuthModeProxy AuthMode = "proxy"
)

// ---------- auth config ----------

// AuthConfig holds all authentication settings.
// It is constructed in the composition root and passed to NewHandler.
type AuthConfig struct {
	Mode AuthMode

	// Local mode fields
	Username       string // plaintext username to match
	HashedPassword []byte // bcrypt hash of the password
	Secret         []byte // HMAC-SHA256 key for signing session cookies

	// Proxy mode fields
	ProxyHeader    string   // header to read username from (default: Remote-User)
	TrustedProxies []string // allowed source IPs; empty = trust all

	// API key (works in any mode except none)
	APIKey string

	// Cookie settings
	CookieName   string
	CookieMaxAge int // seconds
	CookieSecure bool
}

// DefaultAuthConfig returns an AuthConfig with auth disabled.
func DefaultAuthConfig() AuthConfig {
	return AuthConfig{
		Mode:         AuthModeNone,
		CookieName:   "golinks_session",
		CookieMaxAge: 86400, // 24 hours
	}
}

// ---------- context key ----------

type contextKey string

const userContextKey contextKey = "auth_user"

// UserFromContext returns the authenticated username from the request
// context, or empty string if not authenticated.
func UserFromContext(ctx context.Context) string {
	if u, ok := ctx.Value(userContextKey).(string); ok {
		return u
	}
	return ""
}

// ---------- session cookie helpers ----------

// signSession creates a signed session cookie value: base64(username)|base64(HMAC).
func signSession(username string, secret []byte, expiry time.Time) string {
	payload := fmt.Sprintf("%s|%d", username, expiry.Unix())
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	sig := mac.Sum(nil)
	return base64.URLEncoding.EncodeToString([]byte(payload)) + "." +
		base64.URLEncoding.EncodeToString(sig)
}

// verifySession validates a signed session cookie and returns the username.
func verifySession(value string, secret []byte) (string, bool) {
	parts := strings.SplitN(value, ".", 2)
	if len(parts) != 2 {
		return "", false
	}

	payloadBytes, err := base64.URLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", false
	}
	sigBytes, err := base64.URLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", false
	}

	mac := hmac.New(sha256.New, secret)
	mac.Write(payloadBytes)
	if !hmac.Equal(mac.Sum(nil), sigBytes) {
		return "", false
	}

	// Parse payload: "username|unix_timestamp"
	payload := string(payloadBytes)
	sepIdx := strings.LastIndex(payload, "|")
	if sepIdx < 0 {
		return "", false
	}

	username := payload[:sepIdx]
	var expiry int64
	if _, err := fmt.Sscanf(payload[sepIdx+1:], "%d", &expiry); err != nil {
		return "", false
	}
	if time.Now().Unix() > expiry {
		return "", false // expired
	}
	return username, true
}

// GenerateRandomSecret creates a random 32-byte secret for cookie signing.
func GenerateRandomSecret() ([]byte, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}

// ---------- middleware ----------

// RequireAuth returns middleware that protects routes based on AuthConfig.
// For AuthModeNone it passes all requests through.
func RequireAuth(cfg AuthConfig) func(http.Handler) http.Handler {
	// Pre-parse trusted proxy IPs for proxy mode.
	trustedNets := parseTrustedProxies(cfg.TrustedProxies)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch cfg.Mode {
			case AuthModeNone:
				next.ServeHTTP(w, r)
				return

			case AuthModeLocal:
				// Check API key first (for programmatic access)
				if user := checkAPIKey(r, cfg); user != "" {
					ctx := context.WithValue(r.Context(), userContextKey, user)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
				// Check session cookie
				if user := checkSessionCookie(r, cfg); user != "" {
					ctx := context.WithValue(r.Context(), userContextKey, user)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
				handleUnauthorized(w, r)
				return

			case AuthModeProxy:
				// Check API key first
				if user := checkAPIKey(r, cfg); user != "" {
					ctx := context.WithValue(r.Context(), userContextKey, user)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
				// Check trusted proxy header
				if user := checkProxyHeader(r, cfg, trustedNets); user != "" {
					ctx := context.WithValue(r.Context(), userContextKey, user)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
				handleUnauthorized(w, r)
				return

			default:
				http.Error(w, "Server misconfiguration", http.StatusInternalServerError)
			}
		})
	}
}

// ---------- auth check helpers ----------

func checkAPIKey(r *http.Request, cfg AuthConfig) string {
	if cfg.APIKey == "" {
		return ""
	}
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	const prefix = "Bearer "
	if len(auth) > len(prefix) && strings.EqualFold(auth[:len(prefix)], prefix) {
		token := auth[len(prefix):]
		if hmac.Equal([]byte(token), []byte(cfg.APIKey)) {
			return "api-key"
		}
	}
	return ""
}

func checkSessionCookie(r *http.Request, cfg AuthConfig) string {
	cookie, err := r.Cookie(cfg.CookieName)
	if err != nil {
		return ""
	}
	user, ok := verifySession(cookie.Value, cfg.Secret)
	if !ok {
		return ""
	}
	return user
}

func checkProxyHeader(r *http.Request, cfg AuthConfig, trustedNets []*net.IPNet) string {
	header := cfg.ProxyHeader
	if header == "" {
		header = "Remote-User"
	}
	user := r.Header.Get(header)
	if user == "" {
		return ""
	}

	// If trusted proxies are configured, verify the source IP.
	if len(trustedNets) > 0 {
		clientIP := extractIP(r.RemoteAddr)
		if clientIP == nil {
			log.Printf("auth: could not parse remote addr %q", r.RemoteAddr)
			return ""
		}
		trusted := false
		for _, n := range trustedNets {
			if n.Contains(clientIP) {
				trusted = true
				break
			}
		}
		if !trusted {
			log.Printf("auth: untrusted proxy IP %s for header %s", clientIP, header)
			return ""
		}
	}

	return user
}

func handleUnauthorized(w http.ResponseWriter, r *http.Request) {
	// For API requests, return 401 JSON.
	if strings.HasPrefix(r.URL.Path, "/api/") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"authentication required"}`))
		return
	}
	// For browser requests, redirect to login page.
	http.Redirect(w, r, "/login", http.StatusFound)
}

// ---------- IP parsing ----------

func extractIP(remoteAddr string) net.IP {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return net.ParseIP(remoteAddr)
	}
	return net.ParseIP(host)
}

func parseTrustedProxies(proxies []string) []*net.IPNet {
	var nets []*net.IPNet
	for _, p := range proxies {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// If it's not CIDR notation, treat as a single IP.
		if !strings.Contains(p, "/") {
			ip := net.ParseIP(p)
			if ip == nil {
				log.Printf("auth: invalid trusted proxy IP %q", p)
				continue
			}
			bits := 32
			if ip.To4() == nil {
				bits = 128
			}
			p = fmt.Sprintf("%s/%d", ip.String(), bits)
		}
		_, cidr, err := net.ParseCIDR(p)
		if err != nil {
			log.Printf("auth: invalid trusted proxy CIDR %q: %v", p, err)
			continue
		}
		nets = append(nets, cidr)
	}
	return nets
}

// LoggingMiddleware logs the details of each request
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
start := time.Now()

		rw := &loggingResponseWriter{ResponseWriter: w, code: http.StatusOK}
		next.ServeHTTP(rw, r)

		log.Printf("[HTTP] %s %s %s %d %v", r.RemoteAddr, r.Method, r.URL.Path, rw.code, time.Since(start))
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	code int
}

func (rw *loggingResponseWriter) WriteHeader(code int) {
	rw.code = code
	rw.ResponseWriter.WriteHeader(code)
}
