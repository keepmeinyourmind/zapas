package migrations

import "embed"

// FS — SQL-миграции, встроенные в бинарник: отдельно копировать файлы не нужно.
//
//go:embed *.sql
var FS embed.FS
