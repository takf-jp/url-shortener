package background_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/takf-jp/url-shortener/internal/background"
	"github.com/takf-jp/url-shortener/internal/testutil"
	"github.com/takf-jp/url-shortener/repository"
)

func TestAggregator_Run_キャンセルで停止(t *testing.T) {
	db := testutil.NewTestDB(t)
	testutil.TruncateAll(t, db)
	repo := repository.NewClickRepository(db)

	a := background.NewAggregator(repo, 100*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- a.Run(ctx)
	}()

	// 50ms後にキャンセル
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Run did not return after context cancellation")
	}
}
