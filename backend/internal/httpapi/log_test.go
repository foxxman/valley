package httpapi

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"game/backend/internal/auth"
)

func TestLogRequests(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/auth/me", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no", http.StatusUnauthorized)
	})
	mux.HandleFunc("GET /healthcheck", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /swagger/index.html", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := LogRequests(logger, mux)

	me := httptest.NewRequest(http.MethodGet, "/api/auth/me?token=secret", nil)
	handler.ServeHTTP(httptest.NewRecorder(), me)
	health := httptest.NewRequest(http.MethodGet, "/healthcheck", nil)
	handler.ServeHTTP(httptest.NewRecorder(), health)
	swagger := httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil)
	handler.ServeHTTP(httptest.NewRecorder(), swagger)

	logText := buf.String()
	if !strings.Contains(logText, "method=GET") || !strings.Contains(logText, "path=/api/auth/me") || !strings.Contains(logText, "status=401") {
		t.Fatalf("auth log %s", logText)
	}
	if strings.Contains(logText, "secret") || strings.Contains(logText, "token=") {
		t.Fatalf("query leaked into log: %s", logText)
	}
	if !strings.Contains(logText, "path=/healthcheck") || !strings.Contains(logText, "level=DEBUG") {
		t.Fatalf("health log %s", logText)
	}
	if !strings.Contains(logText, "path=/swagger/index.html") || strings.Count(logText, "level=DEBUG") < 2 {
		t.Fatalf("swagger log %s", logText)
	}
}

func TestWriteServiceErrorLogsUnknown(t *testing.T) {
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	recorder := httptest.NewRecorder()
	writeServiceError(recorder, request, errString("db down"))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status %d", recorder.Code)
	}
	if !strings.Contains(buf.String(), "db down") || !strings.Contains(buf.String(), "path=/api/auth/login") {
		t.Fatalf("error log %s", buf.String())
	}
}

func TestWriteServiceErrorSkipsExpectedFailures(t *testing.T) {
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	recorder := httptest.NewRecorder()
	writeServiceError(recorder, request, auth.ErrInvalidCredentials)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", recorder.Code)
	}
	if buf.Len() != 0 {
		t.Fatalf("expected failure was logged: %s", buf.String())
	}
}

type errString string

func (e errString) Error() string { return string(e) }
