// services/user-service/internal/repository/user_repository.go

package repository

import (
	"context"
	"errors"
	"fmt"
	"user-service/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// User Not Found error
var ErrUserNotFound = errors.New("user not found")

// Duplicate User error
var ErrDuplicateUser = errors.New(
	"user with this email or phone number already exists",
)

// User Repository interface
type UserRepository interface {
	Create(
		ctx context.Context, u *model.User,
	) error

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
}

// User Repository implementation
type userRepository struct {
	pool *pgxpool.Pool
}

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

		if errors.As(err, &pgErr) && pgErr.SQLState() == "23505" {
			return ErrDuplicateUser
		}

		return fmt.Errorf("failed to create user: %w", err)
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

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get user by id: %w", err)
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

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}

	if err != nil {
		return nil, fmt.Errorf(
			"failed to get user by email or phone: %w", err,
		)
	}

	return u, nil
}
