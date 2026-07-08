-- +goose Up
CREATE TABLE IF NOT EXISTS user_backups (
    user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    config TEXT NOT NULL DEFAULT '{}',
    state TEXT NOT NULL DEFAULT '{}',
    updated_at DATETIME NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS user_backups;
