// Package main is the entry point for the Svenskt Vin single-binary application.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"text/template"
	"time"

	"github.com/svensktvin/svensktvin/internal/auth"
	"github.com/svensktvin/svensktvin/internal/config"
	"github.com/svensktvin/svensktvin/internal/db"
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
	_ = auth.NewSessionManager(store, cfg.Auth.SessionExpiry) // P2+: used in handler stubs
	_ = auth.NewMagicLinkManager(store)                        // P2+: used in handler stubs
	rateLimiter := auth.NewRateLimiter(cfg.RateLimit.AuthRequests, cfg.RateLimit.AuthWindow)

	// Generate session secret (or load from config)
	sessionSecret := cfg.SessionSecret
	if sessionSecret == "" {
		sessionSecret = randomHex(64)
		slog.Warn("svensktvin: no SESSION_SECRET set, generated random")
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
	// P1-4 stub: full handler implementations come in Phase 2
	mux.HandleFunc("POST /login", auth.RateLimitMiddleware(rateLimiter, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		templates.ExecuteTemplate(w, "auth/login.html", map[string]any{"Message": "P1-4 stub: implement Phase 2"})
	}))
	mux.HandleFunc("GET /login", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		templates.ExecuteTemplate(w, "auth/login.html", map[string]any{"Message": "P2-6 stub"})
	})
	mux.HandleFunc("POST /logout", func(w http.ResponseWriter, r *http.Request) {
		// Clear session cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "session_id",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			Expires:  time.Now().Add(-24 * time.Hour),
		})
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})
	mux.HandleFunc("GET /register", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		templates.ExecuteTemplate(w, "auth/register.html", map[string]any{"Message": "P2-7 stub"})
	})

	// Vineyard routes (require auth)
	// P1-4 stub: full handler implementations come in Phase 2-5
	mux.HandleFunc("GET /vineyard", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})
	mux.HandleFunc("GET /vineyard/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})

	// Static files
	staticFS := http.Dir("static")
	mux.HandleFunc("GET /static/", func(w http.ResponseWriter, r *http.Request) {
		http.FileServer(staticFS).ServeHTTP(w, r)
	})

	// API routes (for HTMX/Alpine JSON endpoints)
	mux.HandleFunc("GET /api/varieties/search", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"query":"","matches":[],"high_confidence":false}`)
	})
	mux.HandleFunc("GET /api/varieties", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
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
	pattern := "templates/**/*.html"
	if mode == "prod" {
		pattern = "templates/**/*.html"
	}

	tmpl, err := template.ParseGlob(pattern)
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	return tmpl, nil
}

// randomHex generates a hex-encoded random string.
func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
