// Package main is the entry point for the Svenskt Vin core API server.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/cors"
	"github.com/svensktvin/core-api/internal/api"
	"github.com/svensktvin/core-api/internal/config"
	"github.com/svensktvin/core-api/internal/db"
)

func main() {
	// Load config
	cfg, err := config.Load("config.yaml")
	if err != nil {
		slog.Error("config: load failed", "err", err)
		os.Exit(1)
	}

	// Setup structured logger
	slog.SetLogLoggerLevel(slog.LevelInfo)

	// Connect to database
	ctx := context.Background()
	store, err := db.NewStore(ctx, cfg.Database.URL)
	if err != nil {
		slog.Error("db: connection failed", "err", err)
		os.Exit(1)
	}
	defer store.Close()

	// Build router
	router := api.NewRouter(store, cfg)

	// CORS middleware
	handler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:4173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Vineyard-Location"},
		AllowCredentials: true,
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.API.Port),
		Handler:      handler.Handler(router),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		slog.Info("core-api: starting server", "port", cfg.API.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server: fatal", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("core-api: shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	server.Shutdown(shutdownCtx)
	slog.Info("core-api: stopped")
}
