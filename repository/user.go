package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type User struct {
	ID           int64  `gorm:"primaryKey"`
	Email        string `gorm:"column:email"`
	PasswordHash string `gorm:"column:password_hash"`
}

func (User) TableName() string { return "users" }

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

var ErrUserExists = errors.New("user already exists")
var ErrUserNotFound = errors.New("user not found")

func (r *UserRepository) Create(ctx context.Context, email, passwordHash string) (*User, error) {
	user := User{
		Email:        email,
		PasswordHash: passwordHash,
	}

	if err := r.db.WithContext(ctx).Create(&user).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return nil, ErrUserExists
		}
		return nil, fmt.Errorf("Create: %w", err)
	}
	return &user, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("FindByEmail: %w", err)
	}
	return &user, nil
}
