-- +goose Up
-- Phase 0 : squelette client (schéma films en Phase 1)
CREATE TABLE IF NOT EXISTS schema_meta (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

INSERT INTO schema_meta (key, value) VALUES ('phase', '0');

-- +goose Down
DROP TABLE IF EXISTS schema_meta;
