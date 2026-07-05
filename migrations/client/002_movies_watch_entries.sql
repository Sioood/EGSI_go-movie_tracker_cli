-- +goose Up
CREATE TABLE IF NOT EXISTS movies (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    title TEXT NOT NULL,
    year INTEGER NOT NULL DEFAULT 0,
    external_id TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_movies_user_id ON movies(user_id);
CREATE INDEX IF NOT EXISTS idx_movies_user_title ON movies(user_id, title);

CREATE TABLE IF NOT EXISTS watch_entries (
    id TEXT PRIMARY KEY,
    movie_id TEXT NOT NULL UNIQUE,
    watched INTEGER NOT NULL DEFAULT 0,
    rating REAL,
    rating_scale INTEGER NOT NULL DEFAULT 10,
    review TEXT NOT NULL DEFAULT '',
    watched_at TEXT,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (movie_id) REFERENCES movies(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_watch_entries_movie_id ON watch_entries(movie_id);

UPDATE schema_meta SET value = '1' WHERE key = 'phase';

-- +goose Down
UPDATE schema_meta SET value = '0' WHERE key = 'phase';
DROP INDEX IF EXISTS idx_watch_entries_movie_id;
DROP TABLE IF EXISTS watch_entries;
DROP INDEX IF EXISTS idx_movies_user_title;
DROP INDEX IF EXISTS idx_movies_user_id;
DROP TABLE IF EXISTS movies;
