package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/takf-jp/url-shortener/internal/auth"
	"github.com/takf-jp/url-shortener/repository"
)

type UserRepository interface {
	Create(ctx context.Context, username, passwordHash string) (*repository.User, error)
	FindByEmail(ctx context.Context, email string) (*repository.User, error)
}

type JWTIssuer interface {
	Issue(userID int64) (string, error)
}

type UserService struct {
	repo      UserRepository
	jwtIssuer JWTIssuer
}

func NewUserService(repo UserRepository, jwtIssuer JWTIssuer) *UserService {
	return &UserService{repo: repo, jwtIssuer: jwtIssuer}
}

var ErrInvalidCredentials = errors.New("invalid email or password")

func (s *UserService) Signup(ctx context.Context, email, password string) (*repository.User, error) {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.repo.Create(ctx, email, hash)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

func (s *UserService) Login(ctx context.Context, email, password string) (string, error) {
	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return "", ErrInvalidCredentials
		}
		return "", fmt.Errorf("find user by email: %w", err)
	}
	if err := auth.VerifyPassword(user.PasswordHash, password); err != nil {
		return "", ErrInvalidCredentials
	}
	token, err := s.jwtIssuer.Issue(user.ID)
	if err != nil {
		return "", fmt.Errorf("issue JWT: %w", err)
	}
	return token, nil
}
