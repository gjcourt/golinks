// Package main is the entry point for the golinks application.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/george/golinks/internal/adapters/http"
	"github.com/george/golinks/internal/adapters/memory"
	"github.com/george/golinks/internal/adapters/postgres"
	"github.com/george/golinks/internal/adapters/sqlite"
	"github.com/george/golinks/internal/app"
	outbound "github.com/george/golinks/internal/ports/outbound"
	"github.com/gorilla/mux"
)

func main() {
	// Configure structured logging to stderr in text format. AGENTS.md
	// declares this as the observability invariant; everything in the
	// process goes through slog.
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))
	slog.SetDefault(logger)

	var repo outbound.LinkRepository
	var userRepo outbound.UserRepository
	var closer func() error

	connStr := os.Getenv("DATABASE_URL")
	switch {
	case strings.HasPrefix(connStr, "postgres://") || strings.HasPrefix(connStr, "postgresql://"):
		pgRepo, err := postgres.NewRepository(connStr)
		if err != nil {
			slog.Error("postgres init failed", "err", err)
			os.Exit(1)
		}
		repo = pgRepo
		userRepo = postgres.NewUserRepository(pgRepo.DB())
		closer = pgRepo.Close
	case strings.HasPrefix(connStr, "sqlite://"):
		path := strings.TrimPrefix(connStr, "sqlite://")
		if path == "" {
			slog.Error("DATABASE_URL=sqlite:// requires a path (e.g. sqlite:///var/lib/golinks/golinks.db)")
			os.Exit(1)
		}
		sqRepo, err := sqlite.NewRepository(path)
		if err != nil {
			slog.Error("sqlite init failed", "path", path, "err", err)
			os.Exit(1)
		}
		repo = sqRepo
		userRepo = sqlite.NewUserRepository(sqRepo.DB())
		closer = sqRepo.Close
	case connStr != "":
		slog.Error("DATABASE_URL has unsupported scheme; expected postgres://, postgresql://, or sqlite://", "value", connStr)
		os.Exit(1)
	default:
		slog.Info("DATABASE_URL not set; using in-memory repository")
		memRepo := memory.NewRepository()
		repo = memRepo
		userRepo = memRepo
		closer = memRepo.Close
	}
	defer func() {
		if closer != nil {
			_ = closer()
		}
	}()

	svc := app.NewLinkService(repo)
	authCfg := buildAuthConfig()

	handler := adapthttp.NewHandler(svc, authCfg, userRepo)
	r := mux.NewRouter()
	r.Use(adapthttp.LoggingMiddleware)
	handler.RegisterRoutes(r)

	port := os.Getenv("GOLINKS_PORT")
	if port == "" {
		port = "8080"
	}

	slog.Info("server starting", "url", "http://localhost:"+port)
	slog.Info("admin ui", "url", "http://localhost:"+port+"/admin")
	slog.Info("auth mode", "mode", authCfg.Mode)

	// In local mode, check whether any users exist and hint at setup.
	if authCfg.Mode == adapthttp.AuthModeLocal {
		n, err := userRepo.CountUsers()
		if err == nil && n == 0 {
			slog.Info("no users found; visit /register to create the first admin", "port", port)
		}
	}

	// Explicit timeouts protect against slow-loris / slow-body DoS — the
	// previous //nolint:gosec http.ListenAndServe left ReadTimeout,
	// WriteTimeout, IdleTimeout, and ReadHeaderTimeout all at zero, so a
	// single client could hold a connection open indefinitely.
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Run the server in a goroutine so we can shut it down gracefully on
	// SIGINT/SIGTERM. Without this, in-flight requests would be cut off
	// abruptly on container stop.
	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-errCh:
		if err != nil {
			slog.Error("server failed", "err", err)
			os.Exit(1)
		}
	case sig := <-stop:
		slog.Info("shutting down", "signal", sig.String())
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			slog.Error("graceful shutdown failed", "err", err)
		}
	}
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
			slog.Error("generate session secret failed", "err", err)
			os.Exit(1)
		}
		cfg.Secret = s
		slog.Warn("no GOLINKS_AUTH_SECRET set; sessions will not survive restarts")
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
