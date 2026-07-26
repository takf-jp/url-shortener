package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/pgconn"
)

type URL struct {
	ID        int64
	Code string
	OriginalURL string
	UserID *int64
}

type URLRepository struct {
	pool *pgxpool.Pool
}

func NewURLRepository(pool *pgxpool.Pool) *URLRepository {
	return &URLRepository{pool: pool}
}

var ErrCodeConflict = errors.New("code already exists")
var ErrNotFound = errors.New("URL not found")

func (r *URLRepository) Create(ctx context.Context, code, originalURL string, userID *int64) (*URL, error) {
	const query = `
		INSERT INTO urls (code, original_url, user_id)
		VALUES ($1, $2, $3)
		RETURNING id, code, original_url
	`

	var url URL
	var pgErr *pgconn.PgError
	err := r.pool.QueryRow(ctx, query, code, originalURL, userID).Scan(&url.ID, &url.Code, &url.OriginalURL)
	if err != nil {
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return nil, ErrCodeConflict
		}
		return nil, fmt.Errorf("Create: %w", err)
	}
	return &url, nil
}

func (r *URLRepository) FindByUserID(ctx context.Context, userID int64) ([]*URL, error) {
	const query = `
		SELECT id, code, original_url
		FROM urls
		WHERE user_id = $1
		ORDER BY id DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("FindByUserID Query: %w", err)
	}
	defer rows.Close()

	var urls []*URL
	for rows.Next() {
		var url URL
		if err := rows.Scan(&url.ID, &url.Code, &url.OriginalURL, &url.UserID); err != nil {
			return nil, fmt.Errorf("FindByUserID: %w", err)
		}
		urls = append(urls, &url)
	}
	if err:= rows.Err(); err != nil {
		return nil, fmt.Errorf("FindByUserID rows.Err: %w", err)
	}
	return urls, nil
}

func (r *URLRepository) FindByCode(ctx context.Context, code string) (*URL, error) {
	const query = `
		SELECT id, code, original_url
		FROM urls
		WHERE code = $1
	`

	var url URL
	err := r.pool.QueryRow(ctx, query, code).Scan(&url.ID, &url.Code, &url.OriginalURL)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("FindByCode: %w", err)
	}
	return &url, nil
}
