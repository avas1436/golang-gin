// services/user-service/internal/repository/refresh_token_repository.go

package repository

import (
	"context"
	stdErrors "errors"
	appErrors "pkg/errors"
	"pkg/postgres"
	"time"
	"user-service/internal/model"

	"github.com/jackc/pgx/v5"
)

type RefreshTokenRepository interface {
	// ساخت
	Create(ctx context.Context, token *model.RefreshToken) error

	// خواندن
	GetByTokenHash(
		ctx context.Context,
		tokenHash string,
	) (
		*model.RefreshToken,
		error,
	)

	// باطل کردن
	Revoke(ctx context.Context, id string) error
}

// ساختار رپوزیتوری توکن رفرش
type refreshTokenRepository struct {
	db postgres.DBTX
}

// ساخت یک رپوزیتوری رفرش توکن جدید
func NewRefreshTokenRepository(db postgres.DBTX) RefreshTokenRepository {

	return &refreshTokenRepository{
		db: db,
	}

}

// ساخت
func (
	r *refreshTokenRepository,
) Create(
	ctx context.Context,
	rt *model.RefreshToken,
) error {

	// اعتبارسنجی ورودی‌ها
	if rt == nil {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"refresh token cannot be nil",
		)
	}

	if rt.UserID == "" {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"user id cannot be empty",
		)
	}

	if rt.TokenHash == "" {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"token hash cannot be empty",
		)
	}

	if rt.ExpiresAt.IsZero() || rt.ExpiresAt.Before(time.Now()) {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"token expiration time must be in the future",
		)
	}

	query := `
		INSERT INTO refresh_tokens (
			user_id, 
			token_hash, 
			expires_at
		)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`

	err := r.db.QueryRow(ctx, query, rt.UserID, rt.TokenHash, rt.ExpiresAt).
		Scan(&rt.ID, &rt.CreatedAt)

	if err != nil {
		// بررسی خطای unique constraint برای token_hash
		var pgErr interface{ SQLState() string }
		if stdErrors.As(err, &pgErr) && pgErr.SQLState() == "23505" {
			return appErrors.New(
				appErrors.KindAlreadyExists,
				"refresh token already exists",
			)
		}

		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to create refresh token in database",
		)
	}

	return nil

}

// خواندن
func (
	r *refreshTokenRepository,
) GetByTokenHash(
	ctx context.Context,
	tokenHash string,
) (
	*model.RefreshToken,
	error,
) {

	// اعتبارسنجی ورودی
	if tokenHash == "" {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"token hash cannot be empty",
		)
	}

	query := `
		SELECT id, user_id, token_hash, expires_at, revoked, created_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`

	rt := &model.RefreshToken{}
	err := r.db.QueryRow(
		ctx,
		query,
		tokenHash,
	).Scan(
		&rt.ID,
		&rt.UserID,
		&rt.TokenHash,
		&rt.ExpiresAt,
		&rt.Revoked,
		&rt.CreatedAt,
	)

	if err != nil {

		if stdErrors.Is(err, pgx.ErrNoRows) {

			return nil, appErrors.New(
				appErrors.KindNotFound,
				"refresh token not found",
			)

		}

		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to get refresh token by hash",
		)

	}

	return rt, nil
}

// باطل کردن
func (
	r *refreshTokenRepository,
) Revoke(
	ctx context.Context,
	id string,
) error {

	// اعتبارسنجی ورودی
	if id == "" {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"token id cannot be empty",
		)
	}

	query := `UPDATE refresh_tokens SET revoked = true WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, id)

	if err != nil {

		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to revoke refresh token",
		)

	}

	if tag.RowsAffected() == 0 {

		return appErrors.New(
			appErrors.KindNotFound,
			"refresh token not found or already revoked",
		)

	}

	return nil
}
