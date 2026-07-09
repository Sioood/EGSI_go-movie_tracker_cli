-- +goose Up
ALTER TABLE movies ADD COLUMN updated_by_device TEXT NOT NULL DEFAULT '';
ALTER TABLE watch_entries ADD COLUMN updated_by_device TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE watch_entries DROP COLUMN updated_by_device;
ALTER TABLE movies DROP COLUMN updated_by_device;
