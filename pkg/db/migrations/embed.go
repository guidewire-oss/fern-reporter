package migrations

import "embed"

// Migrations exposes the embedded migration SQL files.
//
//go:embed *.sql
var EmbeddedMigrations embed.FS
