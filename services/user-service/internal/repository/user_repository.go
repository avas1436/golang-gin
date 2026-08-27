// services/user-service/internal/repository/user_repository.go

package repository

import (
	"context"
	stdErrors "errors"
	appErrors "pkg/errors"
	"user-service/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// User Repository interface
type UserRepository interface {
	Create(ctx context.Context, u *model.User) error

	GetByID(
		ctx context.Context, id string,
	) (
		*model.User, error,
	)

	GetByEmailOrPhone(
		ctx context.Context, emailOrPhone string,
	) (
		*model.User, error,
	)

	Update(ctx context.Context, u *model.User) error
}

// User Repository implementation
type userRepository struct {
	pool *pgxpool.Pool
}

// Create user repository
func NewUserRepository(pool *pgxpool.Pool) UserRepository {

	return &userRepository{
		pool: pool,
	}

}

// Create a new user in the database
func (
	r *userRepository,
) Create(
	ctx context.Context, u *model.User,
) error {

	// اعتبارسنجی اولیه
	if u == nil {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"user cannot be nil",
		)
	}

	if u.PhoneNumber == "" {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"phone number is required",
		)
	}

	query := `
		INSERT INTO users (
			email, phone_number, full_name, password_hash, role
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	err := r.pool.QueryRow(
		ctx,
		query,
		u.Email,
		u.PhoneNumber,
		u.FullName,
		u.PasswordHash,
		u.Role,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)

	if err != nil {
		// 23505 in Postgres means unique constraint violation
		var pgErr interface{ SQLState() string }

		if stdErrors.As(err, &pgErr) && pgErr.SQLState() == "23505" {

			return appErrors.New(
				appErrors.KindAlreadyExists,
				"user with this email or phone number already exists",
			)
		}

		// سایر خطاهای دیتابیس
		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to create user in database",
		)
	}

	return nil
}

// GetByID a user from database
func (
	r *userRepository,
) GetByID(
	ctx context.Context, id string,
) (
	*model.User, error,
) {

	// اعتبارسنجی ID
	if id == "" {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"user id cannot be empty",
		)
	}

	query := `
		SELECT id, email, phone_number, full_name, role, created_at
		FROM users
		WHERE id = $1
	`

	u := &model.User{}

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID,
		&u.Email,
		&u.PhoneNumber,
		&u.FullName,
		&u.Role,
		&u.CreatedAt,
	)

	if err != nil {
		if stdErrors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.New(
				appErrors.KindNotFound,
				"user not found",
			)
		}

		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to get user by id",
		)
	}

	return u, nil
}

// Get by Email or Phone a user from database
func (
	r *userRepository,
) GetByEmailOrPhone(
	ctx context.Context, emailOrPhone string,
) (
	*model.User, error,
) {

	// اعتبارسنجی ورودی
	if emailOrPhone == "" {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"email or phone number cannot be empty",
		)
	}

	query := `
		SELECT 
			id, 
			email, 
			phone_number, 
			full_name, 
			role, 
			created_at
			updated_at
		FROM users
		WHERE email = $1 OR phone_number = $1
	`

	u := &model.User{}
	err := r.pool.QueryRow(ctx, query, emailOrPhone).Scan(
		&u.ID,
		&u.Email,
		&u.PhoneNumber,
		&u.FullName,
		&u.Role,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		if stdErrors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.New(
				appErrors.KindNotFound,
				"user not found",
			)
		}

		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to get user by email or phone",
		)
	}

	return u, nil
}

// Update a user in the database
func (
	r *userRepository,
) Update(
	ctx context.Context,
	u *model.User,
) error {

	if u == nil {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"user cannot be nil",
		)
	}

	query := `
		UPDATE users
		SET 
			email = $1,
			phone_number = $2,
			full_name = $3,
			role = $4,
			updated_at = NOW()
		WHERE id = $5 AND deleted_at IS NULL
		RETURNING updated_at
	`

	result, err := r.pool.Exec(ctx, query,
		u.Email,
		u.PhoneNumber,
		u.FullName,
		u.Role,
		u.ID,
	)

	if err != nil {
		// بررسی unique constraint violation
		var pgErr interface{ SQLState() string }

		if stdErrors.As(err, &pgErr) && pgErr.SQLState() == "23505" {

			return appErrors.New(
				appErrors.KindAlreadyExists,
				"user with this email or phone number already exists",
			)

		}

		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to update user",
		)
	}

	// بررسی اینکه آیا رکوردی آپدیت شده یا نه
	if result.RowsAffected() == 0 {

		return appErrors.New(
			appErrors.KindNotFound,
			"user not found",
		)

	}

	return nil
}
