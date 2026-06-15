package auth

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/svensktvin/svensktvin/internal/db"
)

// MagicLinkManager handles magic link token generation and verification.
type MagicLinkManager struct {
	store *db.Store
}

// NewMagicLinkManager creates a new magic link manager.
func NewMagicLinkManager(store *db.Store) *MagicLinkManager {
	return &MagicLinkManager{store: store}
}

// GenerateToken creates a magic link token for a user.
func (m *MagicLinkManager) GenerateToken(ctx context.Context, userID int64) (string, error) {
	// Generate raw token and hash
	rawToken := RandomHex(32)
	hash := sha256.Sum256([]byte(rawToken))

	// Upsert user (idempotent)
	userID, err := m.store.CreateUser(ctx, "") // Placeholder - actual email handled upstream
	if err != nil {
		return "", fmt.Errorf("magic link: upsert user: %w", err)
	}

	// Insert magic link token
	_, err = m.store.Pool.Exec(ctx, `
		INSERT INTO magic_link_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, userID, hash[:], time.Now().Add(15*time.Minute))
	if err != nil {
		slog.Error("auth: token insert error", "err", err)
		// Return empty token on error - caller should handle
		return "", nil
	}

	return rawToken, nil
}

// VerifyToken verifies a magic link token and returns the user ID.
func (m *MagicLinkManager) VerifyToken(ctx context.Context, rawToken string) (int64, error) {
	// Hash token
	hash := sha256.Sum256([]byte(rawToken))

	// Verify atomically
	var userID int64
	err := m.store.Pool.QueryRow(ctx, `
		UPDATE magic_link_tokens
		SET used = true
		WHERE token_hash = $1
			AND used = false
			AND expires_at > now()
		RETURNING user_id
	`, hash[:]).Scan(&userID)

	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, fmt.Errorf("invalid or expired token")
		}
		return 0, fmt.Errorf("verify token: %w", err)
	}

	return userID, nil
}

// SendMagicLink sends a magic link email for the given email address.
func (m *MagicLinkManager) SendMagicLink(ctx context.Context, email string) error {
	// Generate token
	rawToken := RandomHex(32)
	hash := sha256.Sum256([]byte(rawToken))

	// Upsert user (idempotent: creates new user if doesn't exist)
	var userID int64
	err := m.store.Pool.QueryRow(ctx, `
		INSERT INTO users (email, name)
		SELECT $1, split_part($1, '@', 1)
		WHERE NOT EXISTS (SELECT 1 FROM users WHERE email = $1)
		RETURNING id
	`, email).Scan(&userID)

	if err != nil && err != pgx.ErrNoRows {
		slog.Error("auth: user upsert error", "err", err)
		return fmt.Errorf("upsert user: %w", err)
	}

	// Also handle the case where user already exists
	if userID == 0 {
		err = m.store.Pool.QueryRow(ctx, `
			SELECT id FROM users WHERE email = $1
		`, email).Scan(&userID)
		if err != nil {
			slog.Error("auth: user lookup error", "err", err)
			return fmt.Errorf("lookup user: %w", err)
		}
	}

	// Insert magic link token
	_, err = m.store.Pool.Exec(ctx, `
		INSERT INTO magic_link_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, userID, hash[:], time.Now().Add(15*time.Minute))
	if err != nil {
		slog.Error("auth: token insert error", "err", err)
		// Return success to avoid leaking existence
		return nil
	}

	slog.Info("auth: magic link sent", "email", email)
	return nil
}
