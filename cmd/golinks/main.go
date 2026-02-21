// Package main is the entry point for the golinks application.
package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	adapthttp "github.com/george/golinks/internal/adapter/http"
	"github.com/george/golinks/internal/adapter/memory"
	"github.com/george/golinks/internal/adapter/postgres"
	"github.com/george/golinks/internal/domain"
	"github.com/gorilla/mux"
)

func main() {
	var repo domain.LinkRepository
	var userRepo domain.UserRepository

	connStr := os.Getenv("DATABASE_URL")
	if connStr != "" {
		pgRepo, err := postgres.NewRepository(connStr)
		if err != nil {
			log.Fatalf("Failed to initialize repository: %v", err)
		}
		defer pgRepo.Close() //nolint:errcheck

		repo = pgRepo
		userRepo = postgres.NewUserRepository(pgRepo.DB())
	} else {
		log.Println("DATABASE_URL not set; using in-memory repository")
		memRepo := memory.NewRepository()
		repo = memRepo
		userRepo = memRepo
	}

	svc := domain.NewLinkService(repo)
	authCfg := buildAuthConfig()

	handler := adapthttp.NewHandler(svc, authCfg, userRepo)
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

	// In local mode, check whether any users exist and hint at setup.
	if authCfg.Mode == adapthttp.AuthModeLocal {
		n, err := userRepo.CountUsers()
		if err == nil && n == 0 {
			log.Printf("No users found — visit http://localhost:%s/setup to create the first admin", port)
		}
	}

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

	// Secret for session signing
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
