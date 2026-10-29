package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"game/backend/internal/auth"
	"game/backend/internal/config"
	"game/backend/internal/craft"
	"game/backend/internal/db"
	"game/backend/internal/game"
	"game/backend/internal/gather"
	"game/backend/internal/health"
	"game/backend/internal/httpapi"
	"game/backend/internal/inventory"
	"game/backend/internal/logger"
	"game/backend/internal/mapstate"
	"game/backend/internal/player"
	"game/backend/internal/presence"
	"game/backend/internal/realtime"
)

func main() {
	slog.SetDefault(logger.New())
	if err := run(); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer pool.Close()

	slog.Info("database connected")

	if err := db.Migrate(ctx, pool); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	players := player.NewRepository(pool)
	refreshTokens := auth.NewRefreshRepository(pool)
	authService, err := auth.NewService(players, refreshTokens, auth.ServiceConfig{
		Secret:     []byte(cfg.JWTSecret),
		AccessTTL:  cfg.AccessTTL,
		RefreshTTL: cfg.RefreshTTL,
	})
	if err != nil {
		return fmt.Errorf("auth: %w", err)
	}

	games := game.NewService(game.NewRepository(pool))
	bagRepo := inventory.NewRepository(pool)
	bags := inventory.NewService(bagRepo, games)
	hub := realtime.NewHub()
	router := realtime.NewRouter()
	world := presence.New(hub, games)
	world.Register(router)
	cells := mapstate.NewRepository(pool)
	gathering := gather.NewService(pool, cells, bagRepo, games, world)
	crafting := craft.NewService(pool, bagRepo, games)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthcheck", health.NewHandler(pool))
	httpapi.Register(mux, authService, httpapi.CookieConfig{
		AccessTTL:  cfg.AccessTTL,
		RefreshTTL: cfg.RefreshTTL,
		Secure:     cfg.CookieSecure,
	})
	httpapi.RegisterGames(mux, games, authService)
	httpapi.RegisterInventory(mux, bags, authService)
	httpapi.RegisterMap(mux, gathering, authService, hub)
	httpapi.RegisterCraft(mux, crafting, authService)
	httpapi.RegisterRealtime(mux, authService, hub, router, world.Disconnect)
	httpapi.RegisterSwagger(mux)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.LogRequests(slog.Default(), mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("server listening", "addr", cfg.HTTPAddr)
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		slog.Info("server shutting down")
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server: %w", err)
		}
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}
	return nil
}
