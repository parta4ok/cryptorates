package main

import (
	"database/sql"
	"errors"
	"flag"
	"log"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"

	"cryptorates/tools/migrator"
)

const (
	PgType = "postgres"
)

func main() {
	configPath := flag.String("config", "", "file config path")
	flag.Parse()

	slog.Info("config path", slog.String("path", *configPath))

	parser, err := migrator.NewTaskKeeper(*configPath)
	if err != nil {
		log.Fatalf("failed to create parser: %v", err)
	}

	tasks, err := parser.ParseMigrationTasks(PgType)
	if err != nil {
		log.Fatalf("failed to parse tasks: %v", err)
	}

	for _, task := range tasks {
		if err := runMigration(task); err != nil {
			log.Fatalf("migration failed on service %s with err: %v", task.ServiceName, err)
		}
	}
}

func runMigration(task migrator.MigrationTask) error {
	db, err := sql.Open(PgType, task.ConnectionString)
	if err != nil {
		return err
	}
	defer db.Close() //nolint:errcheck //skip

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(
		task.MigrationPath,
		PgType, driver)
	if err != nil {
		return err
	}
	defer m.Close() //nolint:errcheck //skip

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}
