package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Click struct {
	ID        int64     `gorm:"primaryKey"`
	URLID     int64     `gorm:"column:url_id"`
	AcecessedAt time.Time `gorm:"column:accessed_at;default:now()"`
	IP        string    `gorm:"column:ip"`
	UserAgent string    `gorm:"column:user_agent"`
	Aggregated bool     `gorm:"column:aggregated;default:false"`
}

func (Click) TableName() string { return "url_clicks" }

type ClickRepository struct {
	db *gorm.DB
}


func NewClickRepository(db *gorm.DB) *ClickRepository {
	return &ClickRepository{db: db}
}

func (r *ClickRepository) Insert(ctx context.Context, urlID int64, ip, userAgent string) error {
	click := Click{
		URLID:      urlID,
		IP: ip,
		UserAgent:  userAgent,
	}

	if err := r.db.WithContext(ctx).Create(&click).Error; err != nil {
		return fmt.Errorf("Insert: %w", err)
	}
	return nil
}

// AggregateClickCountsはurl_clicksを集計し、urls.click_countを更新する
func (r *ClickRepository) AggregateClickCounts(ctx context.Context) error {
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

	const markAggregatedQuery = `
		UPDATE url_clicks
		SET aggregated = true
		WHERE aggregated = false AND id <= $1
	`

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var maxID int64
		if err := tx.Raw(`SELECT COALESCE(MAX(id), 0) FROM url_clicks WHERE aggregated = false`).Scan(&maxID).Error; err != nil {
			return fmt.Errorf("failed to get max id: %w", err)
		}

		if err := tx.Exec(aggregateQuery, maxID).Error; err != nil {
			return fmt.Errorf("failed to aggregate click counts: %w", err)
		}

		if err := tx.Exec(markAggregatedQuery, maxID).Error; err != nil {
			return fmt.Errorf("failed to mark clicks as aggregated: %w", err)
		}

		return nil
	})
}
