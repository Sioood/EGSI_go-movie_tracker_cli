-- +goose Up
CREATE TABLE IF NOT EXISTS sync_devices (
    device_id TEXT PRIMARY KEY,
    device_name TEXT NOT NULL DEFAULT '',
    last_seen_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sync_conflicts (
    id TEXT PRIMARY KEY,
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    local_json TEXT NOT NULL,
    remote_json TEXT NOT NULL,
    local_device_id TEXT NOT NULL DEFAULT '',
    remote_device_id TEXT NOT NULL DEFAULT '',
    detected_at TEXT NOT NULL,
    resolved_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_sync_conflicts_open ON sync_conflicts(resolved_at);

ALTER TABLE movies ADD COLUMN updated_by_device TEXT NOT NULL DEFAULT '';
ALTER TABLE watch_entries ADD COLUMN updated_by_device TEXT NOT NULL DEFAULT '';

UPDATE schema_meta SET value = '4' WHERE key = 'phase';

-- +goose Down
UPDATE schema_meta SET value = '3' WHERE key = 'phase';
DROP INDEX IF EXISTS idx_sync_conflicts_open;
DROP TABLE IF EXISTS sync_conflicts;
DROP TABLE IF EXISTS sync_devices;
ALTER TABLE watch_entries DROP COLUMN updated_by_device;
ALTER TABLE movies DROP COLUMN updated_by_device;
