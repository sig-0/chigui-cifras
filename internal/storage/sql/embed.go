package sql

import "embed"

// SchemaFS embeds all SQL schema migration files
//
//go:embed schema/*.sql
var SchemaFS embed.FS
