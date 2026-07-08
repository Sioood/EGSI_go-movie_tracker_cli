-- +goose Up
CREATE TABLE IF NOT EXISTS sync_metadata (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    last_sync_at TEXT,
    last_push_at TEXT,
    last_pull_at TEXT,
    user_id_migrated INTEGER NOT NULL DEFAULT 0
);

INSERT INTO sync_metadata (id) VALUES (1);

CREATE TABLE IF NOT EXISTS sync_pending (
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    operation TEXT NOT NULL,
    PRIMARY KEY (entity_type, entity_id)
);

UPDATE schema_meta SET value = '2' WHERE key = 'phase';

-- +goose Down
UPDATE schema_meta SET value = '1' WHERE key = 'phase';
DROP TABLE IF EXISTS sync_pending;
DROP TABLE IF EXISTS sync_metadata;
