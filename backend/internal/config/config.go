package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	defaultAccessTTL  = 15 * time.Minute
	defaultRefreshTTL = 30 * 24 * time.Hour
	minJWTSecretLen   = 32
)

type Config struct {
	HTTPAddr     string
	DatabaseURL  string
	JWTSecret    string
	AccessTTL    time.Duration
	RefreshTTL   time.Duration
	CookieSecure bool
}

func Load() (Config, error) {
	accessTTL, err := getenvDuration("ACCESS_TTL", defaultAccessTTL)
	if err != nil {
		return Config{}, err
	}
	refreshTTL, err := getenvDuration("REFRESH_TTL", defaultRefreshTTL)
	if err != nil {
		return Config{}, err
	}
	cookieSecure, err := getenvBool("COOKIE_SECURE", false)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		HTTPAddr:     getenv("HTTP_ADDR", ":8080"),
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		JWTSecret:    os.Getenv("JWT_SECRET"),
		AccessTTL:    accessTTL,
		RefreshTTL:   refreshTTL,
		CookieSecure: cookieSecure,
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if len(cfg.JWTSecret) < minJWTSecretLen {
		return Config{}, fmt.Errorf("JWT_SECRET must be at least %d characters", minJWTSecretLen)
	}

	return cfg, nil
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getenvDuration(key string, fallback time.Duration) (time.Duration, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", key)
	}
	return value, nil
}

func getenvBool(key string, fallback bool) (bool, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s must be true or false", key)
	}
	return value, nil
}
