package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSessionCookies(t *testing.T) {
	cfg := CookieConfig{AccessTTL: 15 * time.Minute, RefreshTTL: 30 * 24 * time.Hour, Secure: true}
	rec := httptest.NewRecorder()
	setSessionCookies(rec, cfg, "access-value", "refresh-value")

	cookies := map[string]*http.Cookie{}
	for _, cookie := range rec.Result().Cookies() {
		cookies[cookie.Name] = cookie
	}

	access := cookies[accessCookie]
	refresh := cookies[refreshCookie]
	if access == nil || refresh == nil {
		t.Fatalf("cookies %v", cookies)
	}
	if access.Path != "/" || access.MaxAge != 900 || !access.HttpOnly || !access.Secure || access.SameSite != http.SameSiteLaxMode {
		t.Fatalf("access cookie %+v", access)
	}
	if refresh.Path != "/api/auth" || refresh.MaxAge != 2592000 || refresh.Value != "refresh-value" {
		t.Fatalf("refresh cookie %+v", refresh)
	}

	cleared := httptest.NewRecorder()
	clearSessionCookies(cleared, cfg)
	lines := cleared.Header().Values("Set-Cookie")
	if len(lines) != 2 {
		t.Fatalf("clear headers %v", lines)
	}
	for _, line := range lines {
		if !strings.Contains(line, "Max-Age=0") || !strings.Contains(line, "HttpOnly") {
			t.Fatalf("clear %s", line)
		}
	}
}
