package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ClickRepository struct {
	pool *pgxpool.Pool
}

func NewClickRepository(pool *pgxpool.Pool) *ClickRepository {
	return &ClickRepository{pool: pool}
}

func (r *ClickRepository) Insert(ctx context.Context, urlID int64, ip, userAgent string) error {
	const query = `
		INSERT INTO url_clicks (url_id, ip, user_agent)
		VALUES ($1, $2, $3)
	`
	_, err := r.pool.Exec(ctx, query, urlID, ip, userAgent)
	if err != nil {
		return fmt.Errorf("failed to insert click: %w", err)
	}
	return nil
}

// AggregateClickCountsはurl_clicksを集計し、urls.click_countを更新する
func (r *ClickRepository) AggregateClickCounts(ctx context.Context) error {
	const query = `
		UPDATE urls
		SET click_count = subquery.click_count
		FROM (
			SELECT url_id, COUNT(*) AS click_count
			FROM url_clicks
			GROUP BY url_id
		) AS subquery
		WHERE urls.id = subquery.url_id
	`
	_, err := r.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to aggregate click counts: %w", err)
	}
	return nil
}
