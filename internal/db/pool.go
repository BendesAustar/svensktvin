// Package db provides a connection pool and common query patterns for the Svenskt Vin database.
package db

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store holds the DB connection pool.
type Store struct {
	Pool *pgxpool.Pool
}

// NewStore creates a new database store with the given connection URL.
func NewStore(ctx context.Context, url string) (*Store, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	// Verify the connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	slog.Info("svensktvin: database connected")
	return &Store{Pool: pool}, nil
}

// Close shuts down the connection pool.
func (s *Store) Close() {
	if s.Pool != nil {
		s.Pool.Close()
	}
}
