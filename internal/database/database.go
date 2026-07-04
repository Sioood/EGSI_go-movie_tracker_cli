package database

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

func OpenAndMigrate(dsn string, migrations embed.FS, migrationDir string) (*sql.DB, error) {
	if !isMemoryDSN(dsn) {
		dir := filepath.Dir(sqliteFilePath(dsn))
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("create db dir: %w", err)
			}
		}
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	goose.SetBaseFS(migrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("goose dialect: %w", err)
	}

	if err := goose.Up(db, migrationDir); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

func sqliteFilePath(dsn string) string {
	if len(dsn) > 5 && dsn[:5] == "file:" {
		path := dsn[5:]
		if idx := strings.Index(path, "?"); idx >= 0 {
			path = path[:idx]
		}
		return path
	}
	return dsn
}

func isMemoryDSN(dsn string) bool {
	return dsn == ":memory:" || dsn == "file::memory:"
}
