package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/takf-jp/url-shortener/internal/testutil"
)

func TestURLRepository_Create_正常系(t *testing.T) {
	db := testutil.NewTestDB(t)
	testutil.TruncateAll(t, db)
	repo := NewURLRepository(db)

	u, err := repo.Create(context.Background(), "abc123", "https://example.com", nil)
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

func TestURLRepository_Create_匿名ユーザーはUserIDがnil(t *testing.T) {
	db := testutil.NewTestDB(t)
	testutil.TruncateAll(t, db)
	repo := NewURLRepository(db)

	u, err := repo.Create(context.Background(), "anon123", "https://example.com/anon", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if u.UserID != nil {
		t.Errorf("expected UserID to be nil for anonymous user, got %v", *u.UserID)
	}

	got, err := repo.FindByCode(context.Background(), "anon123")
	if err != nil {
		t.Fatalf("FindByCode failed: %v", err)
	}
	if got.UserID != nil {
		t.Errorf("expected UserID to be nil for anonymous user, got %v", *got.UserID)
	}
}

func TestURLRepository_FindByUserID_自分のURLだけ取得できる(t *testing.T) {
	db := testutil.NewTestDB(t)
	testutil.TruncateAll(t, db)
	repo := NewURLRepository(db)
	ctx := context.Background()

	user1 := User{Email: "user1@example.com", PasswordHash: "hash1"}
	if err := db.Create(&user1).Error; err != nil {
		t.Fatalf("create user1: %v", err)
	}
	user2 := User{Email: "user2@example.com", PasswordHash: "hash2"}
	if err := db.Create(&user2).Error; err != nil {
		t.Fatalf("create user2: %v", err)
	}

	if _, err := repo.Create(ctx, "code1", "https://example.com/1", &user1.ID); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if _, err := repo.Create(ctx, "code2", "https://example.com/2", &user2.ID); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	urls, err := repo.FindByUserID(ctx, user1.ID)
	if err != nil {
		t.Fatalf("FindByUserID failed: %v", err)
	}
	if len(urls) != 1 {
		t.Errorf("expected 1 URL, got %d", len(urls))
	}
	if urls[0].OriginalURL != "https://example.com/1" {
		t.Errorf("expected original URL 'https://example.com/1', got '%s'", urls[0].OriginalURL)
	}
}

func TestURLRepository_FindByCode_存在しない場合_ErrNotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	testutil.TruncateAll(t, db)
	repo := NewURLRepository(db)

	_, err := repo.FindByCode(context.Background(), "no-such-code")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestURLRepository_FindByCode_存在する場合_取得できる(t *testing.T) {
	db := testutil.NewTestDB(t)
	testutil.TruncateAll(t, db)
	repo := NewURLRepository(db)

	_, err := repo.Create(context.Background(), "xyz789", "https://go.dev/", nil)
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
