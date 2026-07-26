package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/takf-jp/url-shortener/repository"
)

type fakeRepo struct {
	createFunc func(ctx context.Context, code, originalURL string, userID *int64) (*repository.URL, error)
	calls int
}

func (f *fakeRepo) Create(ctx context.Context, code, originalURL string, userID *int64) (*repository.URL, error) {
	f.calls++
	return f.createFunc(ctx, code, originalURL, userID)
}

func TestShortenService_Shorten_成功(t *testing.T) {
	repo := &fakeRepo{
		createFunc: func(ctx context.Context, code, originalURL string, userID *int64) (*repository.URL, error) {
			return &repository.URL{Code: code, OriginalURL: originalURL}, nil
		},
	}
	service := NewShortenService(repo)

	u, err := service.Shorten(context.Background(), nil, "https://example.com")
	if err != nil {
		t.Fatalf("Shorten failed: %v", err)
	}
	if u.OriginalURL != "https://example.com" {
		t.Errorf("expected original URL 'https://example.com', got '%s'", u.OriginalURL)
	}
	if repo.calls != 1 {
		t.Errorf("expected 1 call to Create, got %d", repo.calls)
	}
}

func TestShortenService_Shorten_衝突時にリトライ(t *testing.T) {
	callCount := 0
	repo := &fakeRepo{
		createFunc: func(ctx context.Context, code, originalURL string, userID *int64) (*repository.URL, error) {
			callCount++
			if callCount < 2 {
				return nil, repository.ErrCodeConflict
			}
			return &repository.URL{ID: 1, Code: code, OriginalURL: originalURL}, nil
		},
	}
	service := NewShortenService(repo)

	_, err := service.Shorten(context.Background(), nil, "https://example.com")
	if err != nil {
		t.Fatalf("Shorten failed: %v", err)
	}
	if repo.calls != 2 {
		t.Errorf("expected 2 calls to Create due to retry, got %d", repo.calls)
	}
}

func TestShortenService_Shorten_リトライ上限超え_失敗(t *testing.T) {
	repo := &fakeRepo{
		createFunc: func(ctx context.Context, code, originalURL string, userID *int64) (*repository.URL, error) {
			return nil, repository.ErrCodeConflict
		},
	}
	service := NewShortenService(repo)

	_, err := service.Shorten(context.Background(), nil, "https://example.com")
	if err == nil {
		t.Fatalf("expected error due to max retries exceeded, got nil")
	}
	if !errors.Is(err, repository.ErrCodeConflict) {
		t.Fatalf("expected ErrCodeConflict or wrapped error, got %v", err)
    }
    if repo.calls != maxRetry {
        t.Errorf("expected %d calls, got %d", maxRetry, repo.calls)
    }
}

func TestGenerateCode_長さが指定通り(t *testing.T) {
	code, err := generateCode(6)
	if err != nil {
		t.Fatalf("generateCode failed: %v", err)
	}
	if len(code) != 6 {
		t.Errorf("expected length 6, got %d", len(code))
	}
}

func TestGenerateCode_長さ別(t *testing.T) {
    cases := []struct {
        name    string
        length  int
        wantLen int
    }{
        {"6文字", 6, 6},
        {"10文字", 10, 10},
        {"1文字", 1, 1},
    }

    for _, tt := range cases {
        t.Run(tt.name, func(t *testing.T) {
            code, err := generateCode(tt.length)
            if err != nil {
                t.Fatalf("generateCode: %v", err)
            }
            if len(code) != tt.wantLen {
                t.Errorf("got %d, want %d", len(code), tt.wantLen)
            }
        })
    }
}

func TestGenerateCode_文字セット範囲内(t *testing.T) {
	code, err := generateCode(100) //多めに生成
	if err != nil {
		t.Fatalf("generateCode failed: %v", err)
	}
	for _, c := range code {
		if !strings.ContainsRune(charset, c) {
			t.Errorf("character %c is not in charset", c)
		}
	}
}

