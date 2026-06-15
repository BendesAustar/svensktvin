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
	"syscall"
	"time"

	"github.com/svensktvin/svensktvin/internal/auth"
	"github.com/svensktvin/svensktvin/internal/config"
	"github.com/svensktvin/svensktvin/internal/db"
	"github.com/svensktvin/svensktvin/internal/email"
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

	// Generate session secret (or load from config)
	sessionSecret := cfg.SessionSecret
	if sessionSecret == "" {
		slog.Warn("svensktvin: no SESSION_SECRET set")
	}

	// Load templates
	templates, err := loadTemplates(cfg.TemplateMode)
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
	mux.HandleFunc("GET /vineyard", func(w http.ResponseWriter, r *http.Request) {
		user := sessionMgr.SessionFromRequest(r)
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		// Redirect to user's first vineyard (simplified for Phase 2)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})

	// Static files
	staticDir := "static"
	staticFS := http.Dir(staticDir)
	mux.HandleFunc("GET /static/", func(w http.ResponseWriter, r *http.Request) {
		http.FileServer(staticFS).ServeHTTP(w, r)
	})

	// API routes (for HTMX/Alpine JSON endpoints)
	mux.HandleFunc("GET /api/varieties/search", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"query":"","matches":[],"high_confidence":false}`)
	})

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
func loadTemplates(mode string) (*template.Template, error) {
	// Resolve templates directory relative to binary or working directory
	templatePaths := []string{
		"internal/templates/**/*.html",
		"templates/**/*.html",
	}

	var tmpl *template.Template
	var err error

	for _, pattern := range templatePaths {
		tmpl, err = template.ParseGlob(pattern)
		if err == nil {
			slog.Info("svensktvin: templates loaded", "pattern", pattern)
			return tmpl, nil
		}
		slog.Debug("svensktvin: template pattern not found", "pattern", pattern)
	}

	return nil, fmt.Errorf("no templates found in %v", templatePaths)
}

// getTemplateFuncMap returns template helper functions.
func getTemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
	}
}
