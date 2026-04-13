//go:build tools

// tools.go pins build-time and database dependencies so they appear in go.mod
// and go.sum even before application code imports them directly.
// Build with: go build -tags tools .
package main

import (
	_ "github.com/jackc/pgx/v5"
	_ "github.com/oklog/ulid/v2"
	_ "github.com/pressly/goose/v3"
)
