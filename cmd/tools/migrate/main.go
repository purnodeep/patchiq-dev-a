package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

type migrateConfig struct {
	dsn           string
	migrationsDir string
}

func resolveConfig(dbFlag string) (migrateConfig, error) {
	switch dbFlag {
	case "server":
		dsn := os.Getenv("PATCHIQ_DATABASE_URL")
		if dsn == "" {
			dsn = os.Getenv("PATCHIQ_DB_URL") // legacy fallback
		}
		if dsn == "" {
			dsn = "postgres://patchiq:patchiq_dev@localhost:5432/patchiq?sslmode=disable"
		}
		return migrateConfig{dsn: dsn, migrationsDir: "internal/server/store/migrations"}, nil
	case "hub":
		dsn := os.Getenv("PATCHIQ_HUB_DATABASE_URL")
		if dsn == "" {
			dsn = os.Getenv("PATCHIQ_HUB_DB_URL") // legacy fallback
		}
		if dsn == "" {
			dsn = "postgres://patchiq:patchiq_dev@localhost:5432/patchiq_hub?sslmode=disable"
		}
		return migrateConfig{dsn: dsn, migrationsDir: "internal/hub/store/migrations"}, nil
	default:
		return migrateConfig{}, fmt.Errorf("invalid --db value %q: must be \"server\" or \"hub\"", dbFlag)
	}
}

func main() {
	dbFlag := flag.String("db", "server", "target database: server or hub")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "usage: migrate [--db=server|hub] <command>\ncommands: up, down, status, create <name>\n")
		os.Exit(1)
	}
	command := args[0]

	cfg, err := resolveConfig(*dbFlag)
	if err != nil {
		slog.Error("resolve config", "error", err)
		os.Exit(1)
	}

	db, err := sql.Open("pgx", cfg.dsn)
	if err != nil {
		slog.Error("open database", "error", err, "db", *dbFlag)
		os.Exit(1)
	}
	defer db.Close()

	goose.SetBaseFS(nil)
	if err := goose.SetDialect("postgres"); err != nil {
		slog.Error("set dialect", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	switch command {
	case "up":
		err = goose.UpContext(ctx, db, cfg.migrationsDir)
	case "down":
		err = goose.DownContext(ctx, db, cfg.migrationsDir)
	case "status":
		err = goose.StatusContext(ctx, db, cfg.migrationsDir)
	case "create":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "usage: migrate create <name>\n")
			os.Exit(1)
		}
		err = goose.Create(db, cfg.migrationsDir, args[1], "sql")
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q: use up, down, status, or create\n", command)
		os.Exit(1)
	}

	if err != nil {
		slog.Error("migration failed", "command", command, "error", err)
		os.Exit(1)
	}
}
