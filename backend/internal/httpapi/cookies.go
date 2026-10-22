package httpapi

import (
	"net/http"
	"time"
)

const (
	accessCookie  = "access_token"
	refreshCookie = "refresh_token"
	accessPath    = "/"
	refreshPath   = "/api/auth"
)

type CookieConfig struct {
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	Secure     bool
}

func setSessionCookies(w http.ResponseWriter, cfg CookieConfig, access, refresh string) {
	writeCookie(w, cfg, accessCookie, access, accessPath, cfg.AccessTTL)
	writeCookie(w, cfg, refreshCookie, refresh, refreshPath, cfg.RefreshTTL)
}

func clearSessionCookies(w http.ResponseWriter, cfg CookieConfig) {
	writeCookie(w, cfg, accessCookie, "", accessPath, -time.Second)
	writeCookie(w, cfg, refreshCookie, "", refreshPath, -time.Second)
}

func writeCookie(w http.ResponseWriter, cfg CookieConfig, name, value, path string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		Secure:   cfg.Secure,
		SameSite: http.SameSiteLaxMode,
	})
}
