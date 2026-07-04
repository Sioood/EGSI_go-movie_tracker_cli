package database

import "embed"

//go:embed migrations/client/*.sql
var ClientMigrations embed.FS

//go:embed migrations/server/*.sql
var ServerMigrations embed.FS
