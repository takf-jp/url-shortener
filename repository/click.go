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
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // Commit 済みなら何もしない

	// 集計対象の上限を固定する
	var maxID int64
	const maxIDQuery = `
		SELECT COALESCE(MAX(id), 0) FROM url_clicks WHERE aggregated = false
		`
	if err := tx.QueryRow(ctx, maxIDQuery).Scan(&maxID); err != nil {
		return fmt.Errorf("failed to get max id: %w", err)
	}

	const aggregateQuery = `
		UPDATE urls
		SET click_count = urls.click_count + subquery.click_count
		FROM (
			SELECT url_id, COUNT(*) AS click_count
			FROM url_clicks
			WHERE aggregated = false AND id <= $1
			GROUP BY url_id
		) AS subquery
		WHERE urls.id = subquery.url_id
	`
	if _, err := tx.Exec(ctx, aggregateQuery, maxID); err != nil {
		return fmt.Errorf("failed to aggregate click counts: %w", err)
	}

	const markAggregatedQuery = `
		UPDATE url_clicks
		SET aggregated = true
		WHERE aggregated = false AND id <= $1
	`
	if _, err := tx.Exec(ctx, markAggregatedQuery, maxID); err != nil {
		return fmt.Errorf("failed to mark clicks as aggregated: %w", err)
	}

	return tx.Commit(ctx)
}
