package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/takf-jp/url-shortener/repository"
)

type URLRepository interface {
	Create(ctx context.Context, code, originalURL string) (*repository.URL, error)
}

const (
	codeLength = 6
	charset   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	maxRetry = 3
)

type ShortenService struct {
	repo URLRepository
}

func NewShortenService(repo URLRepository) *ShortenService {
	return &ShortenService{repo: repo}
}

// Shortenは URL を短縮する。 衝突時は最大 3 回リトライ。
func (s *ShortenService) Shorten(ctx context.Context, originalURL string) (*repository.URL, error) {
	for i := 0; i < maxRetry; i++ {
		code, err := generateCode(codeLength)
		if err != nil {
			return nil, fmt.Errorf("generateCode: %w", err)
		}

		u, err := s.repo.Create(ctx, code, originalURL)
		if err == nil {
			return u, nil
		}

		if errors.Is(err, repository.ErrCodeConflict) {
			continue // 衝突が発生した場合はリトライ
		}
		return nil, err
	}
	return nil, fmt.Errorf("failed to generate a unique code after %d retries: %w", maxRetry, repository.ErrCodeConflict)
}

func generateCode(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	out := make([]byte, length)
	for i := range b {
		out[i] = charset[int(b[i])%len(charset)]
	}
	return string(out), nil
}
