package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/takf-jp/url-shortener/repository"
)

const (
	codeLength = 6
	charset   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	maxRetry = 3
)

type ShortenService struct {
	repo *repository.URLRepository
}

func NewShortenService(repo *repository.URLRepository) *ShortenService {
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
	return nil, errors.New("failed to generate a unique code after max retries")
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
