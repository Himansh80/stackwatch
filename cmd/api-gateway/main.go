// Package main is the StackWatch API gateway entry point.
// Single binary, modular. Each route is wired in routes.go.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := loadConfig()
	logger.Info("starting api-gateway",
		"http_addr", cfg.HTTPAddr,
		"db_url_set", cfg.DatabaseURL != "",
		"jwt_secret_set", cfg.JWTSecret != "",
		"install_mode", cfg.InstallMode,
	)

	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := connectDatabase(rootCtx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("db connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("database connected")

	issuer := newIssuer(cfg.JWTSecret)

	router := buildRouter(rootCtx, logger, pool, issuer, cfg.InstallMode)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		logger.Info("http listening", "addr", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "err", err)
			cancel()
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	select {
	case s := <-sig:
		logger.Info("signal received", "signal", s.String())
	case <-rootCtx.Done():
		logger.Info("context cancelled")
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown failed", "err", err)
	}
	logger.Info("api-gateway stopped")
}
