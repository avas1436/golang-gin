// services/user-service/internal/repository/refresh_token_repository.go

package repository

import (
	"context"
	"errors"
	"fmt"
	"user-service/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRefreshTokenNotFound = errors.New("refresh token not found")

type RefreshTokenRepository interface {
	// ساخت
	Create(
		ctx context.Context, token *model.RefreshToken,
	) error

	// خواندن
	GetByTokenHash(
		ctx context.Context, tokenHash string,
	) (
		*model.RefreshToken, error,
	)

	// باطل کردن
	Revoke(
		ctx context.Context, id string,
	) error
}

// ساختار رپوزیتوری توکن رفرش
type refreshTokenRepository struct {
	pool *pgxpool.Pool
}

// ساخت یک رپوزیتوری رفرش توکن جدید
func NewRefreshTokenRepository(pool *pgxpool.Pool) RefreshTokenRepository {

	return &refreshTokenRepository{
		pool: pool,
	}

}

// ساخت
func (r *refreshTokenRepository,
) Create(
	ctx context.Context, rt *model.RefreshToken,
) error {

	query := `
		INSERT INTO refresh_tokens (
			user_id, 
			token_hash, 
			expires_at
		)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`

	err := r.pool.QueryRow(ctx, query, rt.UserID, rt.TokenHash, rt.ExpiresAt).
		Scan(&rt.ID, &rt.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create refresh token: %w", err)
	}

	return nil

}

// خواندن
func (
	r *refreshTokenRepository,
) GetByTokenHash(
	ctx context.Context, tokenHash string,
) (*model.RefreshToken, error) {

	query := `
		SELECT id, user_id, token_hash, expires_at, revoked, created_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`

	rt := &model.RefreshToken{}
	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(
		&rt.ID, &rt.UserID, &rt.TokenHash, &rt.ExpiresAt, &rt.Revoked, &rt.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRefreshTokenNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	return rt, nil
}

// باطل کردن
func (
	r *refreshTokenRepository,
) Revoke(
	ctx context.Context, id string,
) error {

	query := `UPDATE refresh_tokens SET revoked = true WHERE id = $1`

	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrRefreshTokenNotFound
	}

	return nil
}
