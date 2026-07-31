-- +goose Up
CREATE TABLE url_clicks (
    id BIGSERIAL PRIMARY KEY,
    url_id BIGINT NOT NULL REFERENCES urls(id) ON DELETE CASCADE,
    accessed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ip VARCHAR(45),
    user_agent TEXT
);

CREATE INDEX idx_url_clicks_url_id ON url_clicks(url_id);

ALTER TABLE urls ADD COLUMN click_count BIGINT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE urls DROP COLUMN click_count;
DROP TABLE url_clicks;
