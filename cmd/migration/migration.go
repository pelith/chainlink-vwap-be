package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"regexp"

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

var errInvalidDatabaseName = errors.New("invalid database name")

func ensureDatabase(ctx context.Context, cfg *config.Config[*migration.Config]) error {
	hostAndPort := net.JoinHostPort(cfg.AppConfig.PostgreSQL.Host, cfg.AppConfig.PostgreSQL.Port)
	connectURI := fmt.Sprintf("postgres://%s:%s@%s/postgres?sslmode=disable", cfg.AppConfig.PostgreSQL.User, cfg.AppConfig.PostgreSQL.Password, hostAndPort)

	db, err := sql.Open("pgx", connectURI)
	if err != nil {
		return fmt.Errorf("connect to postgres: %w", err)
	}
	defer db.Close()

	var exists int

	err = db.QueryRowContext(ctx, "SELECT 1 FROM pg_database WHERE datname = $1", cfg.AppConfig.PostgreSQL.Database).Scan(&exists)
	if err == nil {
		return nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check database: %w", err)
	}

	// Only allow safe identifier for CREATE DATABASE (no user input in config in practice).
	if !regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(cfg.AppConfig.PostgreSQL.Database) {
		return fmt.Errorf("%w: %s", errInvalidDatabaseName, cfg.AppConfig.PostgreSQL.Database)
	}

	_, err = db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", cfg.AppConfig.PostgreSQL.Database))
	if err != nil {
		return fmt.Errorf("create database: %w", err)
	}

	return nil
}

func runMigrate(cfg *config.Config[*migration.Config]) error {
	if err := ensureDatabase(context.Background(), cfg); err != nil {
		return fmt.Errorf("ensure database: %w", err)
	}

	s, err := iofs.New(database.MigrateSQLs, "migrations")
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

	driver, err := migratepgx.WithInstance(db, &migratepgx.Config{})
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
