// Package main provides the admin bootstrap CLI command.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/svensktvin/svensktvin/internal/auth"
	"github.com/svensktvin/svensktvin/internal/db"
)

// adminBootstrapCmd is the "admin bootstrap" subcommand.
type adminBootstrapCmd struct {
	email    string
	password string
}

// Run executes the bootstrap logic: creates first admin if none exists,
// or updates the admin password if one already exists (idempotent).
// Also creates a default vineyard "Min vingård" if none exists.
func (c *adminBootstrapCmd) Run(ctx context.Context, store *db.Store) error {
	// 1. Check if any admin exists
	var adminCount int
	err := store.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM users WHERE is_admin = true
	`).Scan(&adminCount)
	if err != nil {
		return fmt.Errorf("check admin: %w", err)
	}

	if adminCount > 0 {
		// Admin exists — update password
		adminUser, err := store.GetFirstAdmin(ctx)
		if err != nil {
			return fmt.Errorf("find admin: %w", err)
		}

		hash, err := auth.HashPassword(c.password, 12)
		if err != nil {
			return fmt.Errorf("hash password: %w", err)
		}

		err = store.UpdateUserPasswordHash(ctx, adminUser.ID, hash)
		if err != nil {
			return fmt.Errorf("update password: %w", err)
		}

		fmt.Printf("Admin account updated\n")
		fmt.Printf("  Email:    %s\n", adminUser.Email)
		fmt.Printf("\n  Log in at: %s/login\n", os.Getenv("APP_HOST"))
		return nil
	}

	// 2. No admin exists — create one
	hash, err := auth.HashPassword(c.password, 12)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	var userID int64
	err = store.Pool.QueryRow(ctx, `
		INSERT INTO users (email, name, is_admin, password_hash, active)
		VALUES ($1, $2, true, $3, true)
		RETURNING id
	`, c.email, splitName(c.email), hash).Scan(&userID)
	if err != nil {
		return fmt.Errorf("create admin: %w", err)
	}

	fmt.Printf("Admin account created\n")
	fmt.Printf("  Email:    %s\n", c.email)
	fmt.Printf("  Password: %s\n", c.password)
	fmt.Printf("\n  Log in at: %s/login\n", os.Getenv("APP_HOST"))
	fmt.Printf("\n  ⚠  This bootstrap command can be re-run to reset the admin password.\n")

	// 3. Create default vineyard if none exists
	var vineyardCount int
	err = store.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM vineyards WHERE deleted_at IS NULL
	`).Scan(&vineyardCount)
	if err != nil {
		return fmt.Errorf("count vineyards: %w", err)
	}
	if vineyardCount == 0 {
		var vineyardID int64
		err = store.Pool.QueryRow(ctx, `
			INSERT INTO vineyards (name, county, municipality, organic, biodynamic, lat, lon)
			VALUES ('Min vingård', 'Skåne', 'Malmö', false, false, 55.6059, 12.9269)
			RETURNING id
		`).Scan(&vineyardID)
		if err == nil {
			_, err = store.Pool.Exec(ctx, `
				INSERT INTO vineyard_members (vineyard_id, user_id, role)
				VALUES ($1, $2, 'owner')
			`, vineyardID, userID)
			if err == nil {
				fmt.Printf("\nDefault vineyard created: 'Min vingård'\n")
			}
		}
	}

	return nil
}

// splitName extracts a display name from an email address.
func splitName(email string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) == 0 {
		return email
	}
	// Title-case the local part
	name := parts[0]
	if len(name) > 0 {
		name = strings.ToUpper(name[0:1]) + name[1:]
	}
	return name
}

// ExecuteAdminBootstrap parses flags and runs the bootstrap command.
func ExecuteAdminBootstrap(args []string) {
	fs := flag.NewFlagSet("admin bootstrap", flag.ExitOnError)
	email := fs.String("email", "", "Admin email address")
	password := fs.String("password", "", "Admin password (must meet strength requirements)")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Usage: admin bootstrap --email=EMAIL --password=PASSWORD\n")
		os.Exit(1)
	}

	if *email == "" || *password == "" {
		fmt.Fprintf(os.Stderr, "Error: --email and --password are required\n")
		fs.Usage()
		os.Exit(1)
	}

	// Validate password strength
	valid, errors := auth.PasswordStrength(*password)
	if !valid {
		fmt.Fprintf(os.Stderr, "Password does not meet requirements:\n")
		for _, e := range errors {
			fmt.Fprintf(os.Stderr, "  - %s\n", e)
		}
		os.Exit(1)
	}

	// Connect to database
	ctx := context.Background()
	store, err := db.NewStore(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	// Run bootstrap
	if err := (&adminBootstrapCmd{email: *email, password: *password}).Run(ctx, store); err != nil {
		fmt.Fprintf(os.Stderr, "Bootstrap failed: %v\n", err)
		os.Exit(1)
	}
}
