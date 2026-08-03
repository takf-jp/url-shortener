package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type URL struct {
	ID          int64  `gorm:"primaryKey"`
	Code        string `gorm:"column:code"`
	OriginalURL string `gorm:"column:original_url"`
	UserID      *int64 `gorm:"column:user_id"`
	ClickCount  int64  `gorm:"column:click_count"`
}

func (URL) TableName() string { return "urls" }

type URLRepository struct {
	db *gorm.DB
}

func NewURLRepository(db *gorm.DB) *URLRepository {
	return &URLRepository{db: db}
}

var ErrCodeConflict = errors.New("code already exists")
var ErrNotFound = errors.New("URL not found")

func (r *URLRepository) Create(ctx context.Context, code, originalURL string, userID *int64) (*URL, error) {
	url := URL{
		Code:        code,
		OriginalURL: originalURL,
		UserID:      userID,
	}

	if err := r.db.WithContext(ctx).Create(&url).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return nil, ErrCodeConflict
		}
		return nil, fmt.Errorf("Create: %w", err)
	}
	return &url, nil
}

func (r *URLRepository) FindByUserID(ctx context.Context, userID int64) ([]*URL, error) {
	var urls []*URL
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).
		Order("id DESC").
		Find(&urls).Error; err != nil {
		return nil, fmt.Errorf("FindByUserID: %w", err)
	}
	return urls, nil
}

func (r *URLRepository) FindByCode(ctx context.Context, code string) (*URL, error) {
	var url URL
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&url).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("FindByCode: %w", err)
	}
	return &url, nil
}
