-- +goose Up
CREATE TABLE IF NOT EXISTS tmdb_cache (
    tmdb_id INTEGER PRIMARY KEY,
    payload_json TEXT NOT NULL,
    fetched_at TEXT NOT NULL
);

UPDATE schema_meta SET value = '3' WHERE key = 'phase';

-- +goose Down
UPDATE schema_meta SET value = '2' WHERE key = 'phase';
DROP TABLE IF EXISTS tmdb_cache;
