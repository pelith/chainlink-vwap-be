package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net"

	"github.com/golang-migrate/migrate/v4"
	migratepgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"

	"vwap/database"
	"vwap/internal/config"
	"vwap/internal/config/migration"
)

func runSeed(cfg *config.Config[*migration.Config]) error {
	s, err := iofs.New(database.SeedSQLs, fmt.Sprintf("seeds/%s", cfg.Env))
	if err != nil {
		return fmt.Errorf("create iofs source: %w", err)
	}

	hostAndPort := net.JoinHostPort(cfg.AppConfig.PostgreSQL.Host, cfg.AppConfig.PostgreSQL.Port)
	connectURI := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", cfg.AppConfig.PostgreSQL.User, cfg.AppConfig.PostgreSQL.Password, hostAndPort, cfg.AppConfig.PostgreSQL.Database)

	db, err := sql.Open("pgx", connectURI)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer db.Close()

	driver, err := migratepgx.WithInstance(db, &migratepgx.Config{
		MigrationsTable: "seed_migrations",
	})
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", s, cfg.AppConfig.PostgreSQL.Database, driver)
	if err != nil {
		return fmt.Errorf("create migration instance: %w", err)
	}
	defer m.Close()

	if err = m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return nil
		}

		return fmt.Errorf("up: %w", err)
	}

	return nil
}
