-- +goose Up
ALTER TABLE url_clicks ADD COLUMN aggregated BOOLEAN NOT NULL DEFAULT false;
CREATE INDEX idx_url_clicks_aggregated ON url_clicks(aggregated) WHERE aggregated = false;

-- +goose Down
DROP INDEX idx_url_clicks_aggregated;
ALTER TABLE url_clicks DROP COLUMN aggregated;
