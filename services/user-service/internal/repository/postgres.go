// services/user-service/internal/repository/postgres.go

package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPostgresPool make a connection pool to Postgres.
func NewPostgresPool(
	ctx context.Context,
	dsn string,
) (
	*pgxpool.Pool,
	error,
) {

	pool, err := pgxpool.New(ctx, dsn)

	if err != nil {
		return nil, fmt.Errorf("failed to create postgres pool: %w", err)
	}

	// a simple ping to ensure the connection is established, right at the start
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return pool, nil
}
