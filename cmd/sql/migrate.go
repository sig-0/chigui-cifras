package sql

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/sig-0/chigui-cifras/cmd/env"
	dbpkg "github.com/sig-0/chigui-cifras/internal/storage/sql"
)

func newMigrateCmd() *ffcli.Command {
	fs := flag.NewFlagSet("migrate", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "migrate",
		ShortUsage: "sql migrate <file1.sql> [file2.sql ...]",
		LongHelp:   "Run SQL migration files against the database",
		FlagSet:    fs,
		Exec:       execMigrate,
	}
}

func execMigrate(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("at least one migration filename is required")
	}

	// Load .env
	if err := godotenv.Load(); err != nil {
		return fmt.Errorf("unable to load .env vars")
	}

	dsn := os.Getenv(env.Prefix + "_" + env.DatabaseURLSuffix)
	if dsn == "" {
		return fmt.Errorf("%s_%s is required", env.Prefix, env.DatabaseURLSuffix)
	}

	// Open the DB
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}

	// Ping the DB
	if err = db.PingContext(ctx); err != nil {
		return fmt.Errorf("unable to ping DB: %w", err)
	}

	defer func() {
		if err = db.Close(); err != nil {
			fmt.Printf("Unable to gracefully close DB: %s\n", err.Error())
		}
	}()

	for _, name := range args {
		path := fmt.Sprintf("schema/%s", name)

		sqlBytes, err := dbpkg.SchemaFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("unable to read migration %q: %w", name, err)
		}

		fmt.Printf("Running migration %s...\n", name)

		if _, err := db.ExecContext(ctx, string(sqlBytes)); err != nil {
			return fmt.Errorf("unable to run migration %q: %w", name, err)
		}

		fmt.Printf("Migration %q complete\n", name)
	}

	fmt.Println("All migrations complete!")

	return nil
}
