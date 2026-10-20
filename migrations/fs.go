package migrations

import "embed"

// FS содержит SQL-миграции схемы метрик.
//
//go:embed *.sql
var FS embed.FS
