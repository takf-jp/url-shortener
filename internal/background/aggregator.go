package background

import (
	"context"
	"log/slog"
	"time"

	"github.com/takf-jp/url-shortener/internal/logger"
	"github.com/takf-jp/url-shortener/repository"
)

type Aggregator struct {
	repo *repository.ClickRepository
	interval time.Duration
}

func NewAggregator(repo *repository.ClickRepository, interval time.Duration) *Aggregator {
	return &Aggregator{
		repo: repo,
		interval: interval,
	}
}

// Runはctxがキャンセルされるまで、一定間隔でクリック数を集計する
// ctx.Done()が呼ばれると、Runはクリーンに終了する
func (a *Aggregator) Run(ctx context.Context) error {
	log := logger.FromContext(ctx).With(slog.String("component", "aggregator"))

	log.Info("aggregator started", slog.Duration("interval", a.interval))

	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()

	// 起動直後に1回実行
	if err := a.runOnce(ctx, log); err != nil {
		log.Error("first run failed", slog.Any("error", err))
	}

	for {
		select {
		case <-ctx.Done():
			log.Info("aggregator stopped", slog.Any("error", ctx.Err()))
			return ctx.Err()
		case <-ticker.C:
			if err := a.runOnce(ctx, log); err != nil {
				log.Error("aggregation failed", slog.Any("error", err))
				// エラーでも継続（次の周期でリトライ）
			}
		}
	}
}

func (a *Aggregator) runOnce(ctx context.Context, log *slog.Logger) error {
	start := time.Now()

	// 1回の集計に5秒のタイムアウト
	runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := a.repo.AggregateClickCounts(runCtx); err != nil {
		return err
	}
	log.Info("aggregation finished", slog.Duration("elapsed", time.Since(start)))
	return nil
}
