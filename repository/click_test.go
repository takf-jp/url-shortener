package repository

import (
	"context"
	"testing"
	"time"

	"github.com/takf-jp/url-shortener/internal/testutil"
)

func TestClickRepository_Insert_accessed_atが現在時刻(t *testing.T) {
	db := testutil.NewTestDB(t)
	testutil.TruncateAll(t, db)
	urlRepo := NewURLRepository(db)
	repo := NewClickRepository(db)

	u, _ := urlRepo.Create(context.Background(), "click123", "https://example.com/click", nil)

	if err := repo.Insert(context.Background(), u.ID, "127.0.0.1", "test-agent"); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	var accessedAt time.Time
	db.Raw("SELECT accessed_at FROM url_clicks WHERE url_id = ?", u.ID).Scan(&accessedAt)
	if accessedAt.Year() < 2020 {
		t.Errorf("accessed_at looks like a zero value: %v", accessedAt) // 西暦1年ならバグ
	}
}
