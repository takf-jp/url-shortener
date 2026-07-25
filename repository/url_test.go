package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/takf-jp/url-shortener/internal/testutil"
)

func TestURLRepository_Create_正常系(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)
	repo := NewURLRepository(pool)

	u, err := repo.Create(context.Background(), "abc123", "https://example.com")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if u.Code != "abc123" {
		t.Errorf("expected code 'abc123', got '%s'", u.Code)
	}
	if u.OriginalURL != "https://example.com" {
		t.Errorf("expected original URL 'https://example.com', got '%s'", u.OriginalURL)
	}
	if u.ID == 0 {
		t.Errorf("expected non-zero ID, got %d", u.ID)
	}
}

func TestURLRepository_FindByCode_存在しない場合_ErrNotFound(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)
	repo := NewURLRepository(pool)

	_, err := repo.FindByCode(context.Background(), "no-such-code")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestURLRepository_FindByCode_存在する場合_取得できる(t *testing.T) {
	pool := testutil.NewTestPool(t)
	testutil.TruncateAll(t, pool)
	repo := NewURLRepository(pool)

	_, err := repo.Create(context.Background(), "xyz789", "https://go.dev/")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	u, err := repo.FindByCode(context.Background(), "xyz789")
	if err != nil {
		t.Fatalf("FindByCode failed: %v", err)
	}
	if u.OriginalURL != "https://go.dev/" {
		t.Errorf("expected original URL 'https://go.dev/', got '%s'", u.OriginalURL)
	}
}
