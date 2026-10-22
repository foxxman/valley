package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"game/backend/internal/auth"
	"game/backend/internal/player"
)

type fakeAuth struct {
	registerSession auth.Session
	registerErr     error
	loginSession    auth.Session
	loginErr        error
	refreshSession  auth.Session
	refreshErr      error
	logoutErr       error
	current         player.Player
	currentErr      error

	gotLogin    string
	gotPassword string
	gotRefresh  string
	gotAccess   string
}

func (f *fakeAuth) Register(_ context.Context, login, password string) (auth.Session, error) {
	f.gotLogin = login
	f.gotPassword = password
	return f.registerSession, f.registerErr
}

func (f *fakeAuth) Login(_ context.Context, login, password string) (auth.Session, error) {
	f.gotLogin = login
	f.gotPassword = password
	return f.loginSession, f.loginErr
}

func (f *fakeAuth) Refresh(_ context.Context, refreshToken string) (auth.Session, error) {
	f.gotRefresh = refreshToken
	return f.refreshSession, f.refreshErr
}

func (f *fakeAuth) Logout(_ context.Context, refreshToken string) error {
	f.gotRefresh = refreshToken
	return f.logoutErr
}

func (f *fakeAuth) Current(_ context.Context, accessToken string) (player.Player, error) {
	f.gotAccess = accessToken
	return f.current, f.currentErr
}

func newMux(svc AuthService) *http.ServeMux {
	mux := http.NewServeMux()
	Register(mux, svc, CookieConfig{AccessTTL: 15 * time.Minute, RefreshTTL: time.Hour})
	return mux
}

func sessionOf(login string) auth.Session {
	return auth.Session{
		Player:       player.Player{ID: "player-1", Login: login, PasswordHash: "hash-secret"},
		AccessToken:  "access-1",
		RefreshToken: "refresh-1",
	}
}

func TestRegisterHandler(t *testing.T) {
	svc := &fakeAuth{registerSession: sessionOf("fox")}
	rec := request(t, newMux(svc), http.MethodPost, "/api/auth/register", `{"login":"fox","password":"secret12"}`, nil)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	if svc.gotPassword != "secret12" {
		t.Fatalf("password %q", svc.gotPassword)
	}
	assertPlayer(t, rec.Body.Bytes(), "player-1", "fox")
	if strings.Contains(rec.Body.String(), "hash-secret") {
		t.Fatal("response leaked the password hash")
	}
	assertCookie(t, rec, accessCookie, "access-1")
	assertCookie(t, rec, refreshCookie, "refresh-1")
}

func TestAuthErrorMapping(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		body   string
		svc    *fakeAuth
		status int
		code   string
	}{
		{
			name:   "malformed json",
			method: http.MethodPost,
			path:   "/api/auth/register",
			body:   "{",
			svc:    &fakeAuth{},
			status: http.StatusBadRequest,
			code:   "invalid_body",
		},
		{
			name:   "unknown field",
			method: http.MethodPost,
			path:   "/api/auth/login",
			body:   `{"login":"fox","password":"secret12","extra":1}`,
			svc:    &fakeAuth{},
			status: http.StatusBadRequest,
			code:   "invalid_body",
		},
		{
			name:   "login taken",
			method: http.MethodPost,
			path:   "/api/auth/register",
			body:   `{"login":"fox","password":"secret12"}`,
			svc:    &fakeAuth{registerErr: auth.ErrLoginTaken},
			status: http.StatusConflict,
			code:   "login_taken",
		},
		{
			name:   "short password",
			method: http.MethodPost,
			path:   "/api/auth/register",
			body:   `{"login":"fox","password":"short"}`,
			svc:    &fakeAuth{registerErr: auth.ErrInvalidPassword},
			status: http.StatusBadRequest,
			code:   "invalid_password",
		},
		{
			name:   "bad credentials",
			method: http.MethodPost,
			path:   "/api/auth/login",
			body:   `{"login":"fox","password":"secret12"}`,
			svc:    &fakeAuth{loginErr: auth.ErrInvalidCredentials},
			status: http.StatusUnauthorized,
			code:   "invalid_credentials",
		},
		{
			name:   "refresh missing cookie",
			method: http.MethodPost,
			path:   "/api/auth/refresh",
			svc:    &fakeAuth{},
			status: http.StatusUnauthorized,
			code:   "unauthorized",
		},
		{
			name:   "me missing cookie",
			method: http.MethodGet,
			path:   "/api/auth/me",
			svc:    &fakeAuth{},
			status: http.StatusUnauthorized,
			code:   "unauthorized",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rec := request(t, newMux(test.svc), test.method, test.path, test.body, nil)
			if rec.Code != test.status {
				t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
			}
			assertErrorCode(t, rec.Body.Bytes(), test.code)
		})
	}
}

func TestRefreshLogoutAndMe(t *testing.T) {
	svc := &fakeAuth{
		refreshSession: sessionOf("fox"),
		current:        player.Player{ID: "player-1", Login: "fox", PasswordHash: "hash-secret"},
	}
	mux := newMux(svc)

	refreshReq := request(t, mux, http.MethodPost, "/api/auth/refresh", "", map[string]string{
		refreshCookie: "old-refresh",
	})
	if refreshReq.Code != http.StatusOK {
		t.Fatalf("refresh %d %s", refreshReq.Code, refreshReq.Body.String())
	}
	if svc.gotRefresh != "old-refresh" {
		t.Fatalf("refresh token %q", svc.gotRefresh)
	}
	assertCookie(t, refreshReq, accessCookie, "access-1")

	logoutReq := request(t, mux, http.MethodPost, "/api/auth/logout", "", map[string]string{
		refreshCookie: "old-refresh",
	})
	if logoutReq.Code != http.StatusNoContent {
		t.Fatalf("logout %d", logoutReq.Code)
	}
	if logoutReq.Body.Len() != 0 {
		t.Fatalf("logout body %q", logoutReq.Body.String())
	}
	cleared := 0
	for _, line := range logoutReq.Header().Values("Set-Cookie") {
		if strings.Contains(line, "Max-Age=0") {
			cleared++
		}
	}
	if cleared != 2 {
		t.Fatalf("cleared %d headers %v", cleared, logoutReq.Header().Values("Set-Cookie"))
	}

	emptyLogout := request(t, mux, http.MethodPost, "/api/auth/logout", "", nil)
	if emptyLogout.Code != http.StatusNoContent {
		t.Fatalf("empty logout %d", emptyLogout.Code)
	}

	meReq := request(t, mux, http.MethodGet, "/api/auth/me", "", map[string]string{
		accessCookie: "access-1",
	})
	if meReq.Code != http.StatusOK {
		t.Fatalf("me %d %s", meReq.Code, meReq.Body.String())
	}
	if svc.gotAccess != "access-1" {
		t.Fatalf("access %q", svc.gotAccess)
	}
	assertPlayer(t, meReq.Body.Bytes(), "player-1", "fox")
	if strings.Contains(meReq.Body.String(), "hash-secret") {
		t.Fatal("me leaked the password hash")
	}
}

func request(t *testing.T, handler http.Handler, method, path, body string, cookies map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	for name, value := range cookies {
		req.AddCookie(&http.Cookie{Name: name, Value: value})
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func assertPlayer(t *testing.T, body []byte, id, login string) {
	t.Helper()
	var got playerBody
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != id || got.Login != login {
		t.Fatalf("player %+v", got)
	}
}

func assertErrorCode(t *testing.T, body []byte, code string) {
	t.Helper()
	var got errorBody
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode %s: %v", body, err)
	}
	if got.Error.Code != code || got.Error.Message == "" {
		t.Fatalf("error %+v", got.Error)
	}
}

func assertCookie(t *testing.T, rec *httptest.ResponseRecorder, name, value string) {
	t.Helper()
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == name {
			if cookie.Value != value || !cookie.HttpOnly {
				t.Fatalf("cookie %+v", cookie)
			}
			return
		}
	}
	t.Fatalf("missing cookie %s", name)
}
