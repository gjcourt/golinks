package adapthttp

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoggingMiddleware(t *testing.T) {
	t.Parallel()
	// Create a dummy handler
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("OK"))
	})

	// Wrap it
	handler := LoggingMiddleware(nextHandler)

	// Capture slog output by swapping the default logger for the duration
	// of the test. LoggingMiddleware uses slog.Info on the default logger.
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	defer slog.SetDefault(prev)

	req := httptest.NewRequest("GET", "/test-path", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Check response
	if w.Code != http.StatusTeapot {
		t.Errorf("Expected status %d, got %d", http.StatusTeapot, w.Code)
	}

	// Check log
	logOutput := buf.String()
	if !strings.Contains(logOutput, "GET") ||
		!strings.Contains(logOutput, "/test-path") ||
		!strings.Contains(logOutput, "418") {
		t.Errorf("Log output missing expected fields. Got: %s", logOutput)
	}
}
