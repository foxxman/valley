package httpapi

import (
	"bufio"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"
)

type statusWriter struct {
	http.ResponseWriter
	status  int
	written bool
}

func (w *statusWriter) WriteHeader(status int) {
	if w.written {
		return
	}
	w.status = status
	w.written = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(body []byte) (int, error) {
	if !w.written {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

func (w *statusWriter) Flush() {
	if !w.written {
		w.WriteHeader(http.StatusOK)
	}
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	conn, reader, err := hijacker.Hijack()
	if err != nil {
		return nil, nil, err
	}
	w.status = http.StatusSwitchingProtocols
	w.written = true
	// http.Server deadlines stay on the hijacked conn and would close a live socket.
	_ = conn.SetDeadline(time.Time{})
	return conn, reader, nil
}

func (w *statusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// LogRequests writes one line per request: method, path, status, duration.
// A successful healthcheck stays at debug so probes do not flood the log.
func LogRequests(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		writer := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(writer, r)

		level := requestLogLevel(r.URL.Path, writer.status)
		logger.Log(r.Context(), level, "request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", writer.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

func requestLogLevel(path string, status int) slog.Level {
	if status >= http.StatusInternalServerError {
		return slog.LevelError
	}
	if status >= http.StatusBadRequest {
		return slog.LevelWarn
	}
	if path == "/healthcheck" || strings.HasPrefix(path, "/swagger") {
		return slog.LevelDebug
	}
	return slog.LevelInfo
}
