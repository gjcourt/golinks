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

	adapthttp "github.com/george/golinks/internal/adapters/http"
	"github.com/george/golinks/internal/adapters/memory"
	"github.com/george/golinks/internal/adapters/postgres"
	"github.com/george/golinks/internal/adapters/sqlite"
	"github.com/george/golinks/internal/app"
	"github.com/george/golinks/internal/ports/outbound"
	"github.com/gorilla/mux"
)

func main() {
	// Configure structured logging to stderr in text format. AGENTS.md
	// declares this as the observability invariant; everything in the
	// process goes through slog.
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))
	slog.SetDefault(logger)

	repo, userRepo, closer := openRepositories(os.Getenv("DATABASE_URL"))
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

	srv := newServer(port, r)
	runServer(srv)
}

// openRepositories selects a repository implementation based on the
// connection string scheme. It returns the link repository, user
// repository, and a closer function (which may be nil).
func openRepositories(connStr string) (outbound.LinkRepository, outbound.UserRepository, func() error) {
	switch {
	case strings.HasPrefix(connStr, "postgres://") || strings.HasPrefix(connStr, "postgresql://"):
		pgRepo, err := postgres.NewRepository(connStr)
		if err != nil {
			slog.Error("postgres init failed", "err", err)
			os.Exit(1)
		}
		return pgRepo, postgres.NewUserRepository(pgRepo.DB()), pgRepo.Close
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
		return sqRepo, sqlite.NewUserRepository(sqRepo.DB()), sqRepo.Close
	case connStr != "":
		slog.Error("DATABASE_URL has unsupported scheme; expected postgres://, postgresql://, or sqlite://", "value", connStr)
		os.Exit(1)
		return nil, nil, nil // unreachable
	default:
		slog.Info("DATABASE_URL not set; using in-memory repository")
		memRepo := memory.NewRepository()
		return memRepo, memRepo, memRepo.Close
	}
}

// newServer builds an http.Server with explicit timeouts to protect
// against slow-loris / slow-body DoS — the previous //nolint:gosec
// http.ListenAndServe left ReadTimeout, WriteTimeout, IdleTimeout, and
// ReadHeaderTimeout all at zero, so a single client could hold a
// connection open indefinitely.
func newServer(port string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

// runServer starts srv in a goroutine and blocks until either the
// server fails or a SIGINT/SIGTERM signal triggers a graceful shutdown.
// Without this, in-flight requests would be cut off abruptly on
// container stop.
func runServer(srv *http.Server) {
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
