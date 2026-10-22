package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"game/backend/internal/auth"
	"game/backend/internal/player"
)

type AuthService interface {
	Register(ctx context.Context, login, password string) (auth.Session, error)
	Login(ctx context.Context, login, password string) (auth.Session, error)
	Refresh(ctx context.Context, refreshToken string) (auth.Session, error)
	Logout(ctx context.Context, refreshToken string) error
	Current(ctx context.Context, accessToken string) (player.Player, error)
}

type credentials struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type playerBody struct {
	ID    string `json:"id"`
	Login string `json:"login"`
}

func Register(mux *http.ServeMux, svc AuthService, cookies CookieConfig) {
	h := &authHandler{svc: svc, cookies: cookies}
	mux.HandleFunc("POST /api/auth/register", h.register)
	mux.HandleFunc("POST /api/auth/login", h.login)
	mux.HandleFunc("POST /api/auth/refresh", h.refresh)
	mux.HandleFunc("POST /api/auth/logout", h.logout)
	mux.HandleFunc("GET /api/auth/me", h.me)
}

type authHandler struct {
	svc     AuthService
	cookies CookieConfig
}

func (h *authHandler) register(w http.ResponseWriter, r *http.Request) {
	body, err := readCredentials(w, r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body")
		return
	}
	session, err := h.svc.Register(r.Context(), body.Login, body.Password)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	h.writeSession(w, http.StatusCreated, session)
}

func (h *authHandler) login(w http.ResponseWriter, r *http.Request) {
	body, err := readCredentials(w, r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body")
		return
	}
	session, err := h.svc.Login(r.Context(), body.Login, body.Password)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	h.writeSession(w, http.StatusOK, session)
}

func (h *authHandler) refresh(w http.ResponseWriter, r *http.Request) {
	raw := cookieValue(r, refreshCookie)
	if raw == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	session, err := h.svc.Refresh(r.Context(), raw)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	h.writeSession(w, http.StatusOK, session)
}

func (h *authHandler) logout(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Logout(r.Context(), cookieValue(r, refreshCookie)); err != nil {
		writeServiceError(w, r, err)
		return
	}
	clearSessionCookies(w, h.cookies)
	w.WriteHeader(http.StatusNoContent)
}

func (h *authHandler) me(w http.ResponseWriter, r *http.Request) {
	raw := cookieValue(r, accessCookie)
	if raw == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	current, err := h.svc.Current(r.Context(), raw)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, playerBody{ID: current.ID, Login: current.Login})
}

func (h *authHandler) writeSession(w http.ResponseWriter, status int, session auth.Session) {
	setSessionCookies(w, h.cookies, session.AccessToken, session.RefreshToken)
	writeJSON(w, status, playerBody{ID: session.Player.ID, Login: session.Player.Login})
}

func readCredentials(w http.ResponseWriter, r *http.Request) (credentials, error) {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var body credentials
	if err := decoder.Decode(&body); err != nil {
		return credentials{}, err
	}
	err := decoder.Decode(&struct{}{})
	if errors.Is(err, io.EOF) {
		return body, nil
	}
	if err == nil {
		return credentials{}, errors.New("trailing json")
	}
	return credentials{}, err
}

func cookieValue(r *http.Request, name string) string {
	cookie, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}
