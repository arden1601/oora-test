// Package migrations carries the SQL files inside the server binary, so the
// one command that starts the server also brings the database up to date.
package migrations

import "embed"

// Files holds every .sql file in this folder. They are applied in name order.
//
//go:embed *.sql
var Files embed.FS
