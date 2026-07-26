package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/pgconn"
)

type User struct {
	ID          int64
	Email        string
	PasswordHash string
}

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

var ErrUserExists = errors.New("user already exists")
var ErrUserNotFound = errors.New("user not found")

func (r *UserRepository) Create(ctx context.Context, email, passwordHash string) (*User, error) {
	const query = `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id, email, password_hash
	`

	var user User
	err := r.pool.QueryRow(ctx, query, email, passwordHash).Scan(&user.ID, &user.Email, &user.PasswordHash)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return nil, ErrUserExists
		}
		return nil, fmt.Errorf("Create: %w", err)
	}
	return &user, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	const query = `
		SELECT id, email, password_hash
		FROM users
		WHERE email = $1
	`

	var user User
	err := r.pool.QueryRow(ctx, query, email).Scan(&user.ID, &user.Email, &user.PasswordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("FindByEmail: %w", err)
	}
	return &user, nil
}
