package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSwaggerUI(t *testing.T) {
	mux := http.NewServeMux()
	RegisterSwagger(mux)

	index := httptest.NewRecorder()
	mux.ServeHTTP(index, httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil))
	if index.Code != http.StatusOK || !strings.Contains(index.Body.String(), "SwaggerUIBundle") {
		t.Fatalf("index status %d", index.Code)
	}
	if !strings.Contains(index.Body.String(), "withCredentials") {
		t.Fatalf("ui does not send cookies")
	}

	spec := httptest.NewRecorder()
	mux.ServeHTTP(spec, httptest.NewRequest(http.MethodGet, "/swagger/doc.json", nil))
	if spec.Code != http.StatusOK {
		t.Fatalf("spec status %d", spec.Code)
	}
	body := spec.Body.String()
	for _, part := range []string{"/api/auth/login", "/api/games/{id}", "/api/ws", "worldSeed", "AccessCookie", "\"player\""} {
		if !strings.Contains(body, part) {
			t.Fatalf("spec missing %s", part)
		}
	}

	redirect := httptest.NewRecorder()
	mux.ServeHTTP(redirect, httptest.NewRequest(http.MethodGet, "/swagger", nil))
	if redirect.Code != http.StatusFound || redirect.Header().Get("Location") != "/swagger/index.html" {
		t.Fatalf("redirect %d %s", redirect.Code, redirect.Header().Get("Location"))
	}
}
