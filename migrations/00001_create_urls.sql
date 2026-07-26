-- +goose Up
CREATE TABLE urls (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(16) UNIQUE NOT NULL,
    original_url TEXT NOT NULL,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_urls_code ON urls(code);

-- +goose Down
DROP TABLE urls;
