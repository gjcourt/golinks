// Package main is the entry point for the golinks application.
package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	adapthttp "github.com/george/golinks/internal/adapter/http"
	"github.com/george/golinks/internal/adapter/postgres"
	"github.com/george/golinks/internal/domain"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("DATABASE_URL is required")
	}

	repo, err := postgres.NewRepository(connStr)
	if err != nil {
		log.Fatalf("Failed to initialize repository: %v", err)
	}
	defer repo.Close() //nolint:errcheck

	svc := domain.NewLinkService(repo)
	authCfg := buildAuthConfig()

	handler := adapthttp.NewHandler(svc, authCfg)
	r := mux.NewRouter()
	r.Use(adapthttp.LoggingMiddleware)
	handler.RegisterRoutes(r)

	port := os.Getenv("GOLINKS_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("GoLinks server starting on http://localhost:%s", port)
	log.Printf("Admin UI available at http://localhost:%s/admin", port)
	log.Printf("Auth mode: %s", authCfg.Mode)
	//nolint:gosec // ignoring timeout constraint for simple server
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func buildAuthConfig() adapthttp.AuthConfig {
	cfg := adapthttp.DefaultAuthConfig()

	mode := os.Getenv("GOLINKS_AUTH_MODE")
	switch strings.ToLower(mode) {
	case "local":
		cfg.Mode = adapthttp.AuthModeLocal
	case "proxy":
		cfg.Mode = adapthttp.AuthModeProxy
	default:
		cfg.Mode = adapthttp.AuthModeNone
		return cfg
	}

	// Secret for session signing (local mode)
	if secret := os.Getenv("GOLINKS_AUTH_SECRET"); secret != "" {
		cfg.Secret = []byte(secret)
	} else if cfg.Mode == adapthttp.AuthModeLocal {
		s, err := adapthttp.GenerateRandomSecret()
		if err != nil {
			log.Fatalf("Failed to generate session secret: %v", err)
		}
		cfg.Secret = s
		log.Println("WARNING: No GOLINKS_AUTH_SECRET set — sessions will not survive restarts")
	}

	// Local mode credentials
	if cfg.Mode == adapthttp.AuthModeLocal {
		cfg.Username = os.Getenv("GOLINKS_AUTH_USERNAME")
		if cfg.Username == "" {
			log.Fatal("GOLINKS_AUTH_USERNAME is required in local auth mode")
		}
		password := os.Getenv("GOLINKS_AUTH_PASSWORD")
		if password == "" {
			log.Fatal("GOLINKS_AUTH_PASSWORD is required in local auth mode")
		}
		hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			log.Fatalf("Failed to hash password: %v", err)
		}
		cfg.HashedPassword = hashed
	}

	// Proxy mode settings
	if cfg.Mode == adapthttp.AuthModeProxy {
		if header := os.Getenv("GOLINKS_AUTH_HEADER"); header != "" {
			cfg.ProxyHeader = header
		} else {
			cfg.ProxyHeader = "Remote-User"
		}
		if proxies := os.Getenv("GOLINKS_AUTH_TRUSTED_PROXIES"); proxies != "" {
			cfg.TrustedProxies = strings.Split(proxies, ",")
		}
	}

	// API key (works in both local and proxy modes)
	cfg.APIKey = os.Getenv("GOLINKS_API_KEY")

	// Cookie settings
	if os.Getenv("GOLINKS_COOKIE_SECURE") == "true" {
		cfg.CookieSecure = true
	}

	return cfg
}
