// Package main is the entry point for the Svenskt Vin single-binary application.
package main

import (
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/svensktvin/svensktvin/internal/auth"
	"github.com/svensktvin/svensktvin/internal/config"
	"github.com/svensktvin/svensktvin/internal/db"
	"github.com/svensktvin/svensktvin/internal/email"
	"github.com/svensktvin/svensktvin/internal/handlers/api"
	"github.com/svensktvin/svensktvin/internal/handlers/pages"
)

func main() {
	// Load configuration
	cfg, err := config.Load("config.yaml")
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	slog.Info("svensktvin: starting server", "port", cfg.Port)

	// Connect to database
	ctx := context.Background()
	store, err := db.NewStore(ctx, cfg.Database.URL)
	if err != nil {
		slog.Error("failed to connect to database", "err", err)
		os.Exit(1)
	}
	defer store.Close()

	// Initialize auth components
	sessionMgr := auth.NewSessionManager(store, cfg.Auth.SessionExpiry)
	magicLinkMgr := auth.NewMagicLinkManager(store)
	rateLimiter := auth.NewRateLimiter(cfg.RateLimit.AuthRequests, cfg.RateLimit.AuthWindow)

	// Initialize email sender
	emailSender := email.NewSender(email.Config{
		Host: cfg.SMTP.Host,
		Port: cfg.SMTP.Port,
		User: cfg.SMTP.User,
		Pass: cfg.SMTP.Pass,
		From: cfg.SMTP.From,
	})

	// Initialize auth handlers
	authHandler := pages.NewAuthHandler(store, sessionMgr, magicLinkMgr, rateLimiter, cfg, emailSender)

	// Initialize vineyard handlers
	vineyardHandler := pages.NewVineyardHandler(store, sessionMgr)

	// Initialize harvest handlers
	harvestHandler := pages.NewHarvestHandler(store, sessionMgr)
	harvestLockHandler := pages.NewHarvestLockHandler(store, sessionMgr)

	// Initialize account API handlers
	accountHandler := api.NewAccountHandler(store, sessionMgr)

	// Initialize API handlers
	varietySearchHandler := api.NewVarietySearchHandler(store)
	geoReverseHandler := api.NewGeoReverseHandler()

	// Generate session secret (or load from config)
	sessionSecret := cfg.SessionSecret
	if sessionSecret == "" {
		slog.Warn("svensktvin: no SESSION_SECRET set")
	}

	// Load templates
	templates, err := loadTemplates(getTemplateFuncMap())
	if err != nil {
		slog.Error("failed to load templates", "err", err)
		os.Exit(1)
	}

	// Build router
	mux := http.NewServeMux()

	// Health check (public, no auth)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","version":"1.0.0","database":"connected"}`)
	})

	// Auth routes (public, rate-limited)
	mux.HandleFunc("GET /login", authHandler.HandleLoginGET(templates))
	mux.HandleFunc("POST /login", auth.RateLimitMiddleware(rateLimiter, authHandler.HandleLoginPOST(templates)))
	mux.HandleFunc("POST /logout", authHandler.HandleLogoutPOST(templates))
	mux.HandleFunc("GET /register", authHandler.HandleRegisterGET(templates))
	mux.HandleFunc("POST /register", auth.RateLimitMiddleware(rateLimiter, authHandler.HandleRegisterPOST(templates)))
	mux.HandleFunc("GET /auth/forgot-password", authHandler.HandleForgotPasswordGET(templates))
	mux.HandleFunc("POST /auth/forgot-password", auth.RateLimitMiddleware(rateLimiter, authHandler.HandleForgotPasswordPOST(templates)))
	mux.HandleFunc("GET /auth/set-password", authHandler.HandleSetPasswordGET(templates))
	mux.HandleFunc("POST /auth/set-password", authHandler.HandleSetPasswordPOST(templates))
	mux.HandleFunc("GET /invite/confirm", authHandler.HandleInviteConfirmGET(templates))
	mux.HandleFunc("POST /invite/confirm", auth.RateLimitMiddleware(rateLimiter, authHandler.HandleInviteConfirmPOST(templates)))

	// Vineyard routes (require auth)
	mux.HandleFunc("GET /vineyard", vineyardHandler.HandleLandingGET(templates))
	mux.HandleFunc("GET /vineyard/", vineyardHandler.HandleVineyardGET(templates))
	mux.HandleFunc("GET /vineyard/benchmark", vineyardHandler.HandleBenchmarkGET(templates))

	// Harvest POST routes (require auth) — locked by method+path
	mux.HandleFunc("POST /vineyard/", vineyardHandler.HandleVineyardPOST(templates, harvestHandler, harvestLockHandler))

	// Account API routes (require auth)
	mux.HandleFunc("GET /account/export", accountHandler.HandleAccountExportGET)
	mux.HandleFunc("POST /account/delete", accountHandler.HandleAccountDeletePOST)

	// Static files
	staticDir := "static"
	staticFS := http.Dir(staticDir)
	mux.HandleFunc("GET /static/", func(w http.ResponseWriter, r *http.Request) {
		http.FileServer(staticFS).ServeHTTP(w, r)
	})

	// API routes (for HTMX/Alpine JSON endpoints)
	mux.HandleFunc("GET /api/varieties/search", varietySearchHandler.HandleGET)
	mux.HandleFunc("POST /api/geo/reverse", geoReverseHandler.HandlePOST)

	// Create server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigch := make(chan os.Signal, 1)
		signal.Notify(sigch, syscall.SIGINT, syscall.SIGTERM)
		<-sigch

		slog.Info("svensktvin: shutting down server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("server shutdown error", "err", err)
		}
	}()

	// Start server
	slog.Info("svensktvin: listening", "addr", server.Addr)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}

	slog.Info("svensktvin: server stopped")
}

// loadTemplates loads all Go HTML templates from the templates directory.
func loadTemplates(funcMap template.FuncMap) (*template.Template, error) {
	// Try multiple template directories
	templateDirs := []string{
		"internal/templates",
		"templates",
	}

	for _, dir := range templateDirs {
		tmpl, err := loadTemplatesFromDir(dir, funcMap)
		if err != nil {
			slog.Debug("svensktvin: template dir not found", "dir", dir, "err", err)
			continue
		}
		slog.Info("svensktvin: templates loaded", "dir", dir, "count", len(tmpl.Templates()))
		return tmpl, nil
	}

	return nil, fmt.Errorf("no templates found in %v", templateDirs)
}

// loadTemplatesFromDir recursively loads all .html templates from a directory.
func loadTemplatesFromDir(dir string, funcMap template.FuncMap) (*template.Template, error) {
	var paths []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".html") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk templates dir %s: %w", dir, err)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no .html files found in %s", dir)
	}

	// Load templates with the func map
	tmpl := template.New("")
	tmpl = tmpl.Funcs(funcMap)
	_, err = tmpl.ParseFiles(paths...)
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	return tmpl, nil
}

// getTemplateFuncMap returns template helper functions.
func getTemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
	}
}
