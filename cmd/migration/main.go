package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"vwap/internal/config"
	"vwap/internal/config/migration"
)

func main() {
	env := os.Getenv("ENV")
	if env == "" {
		env = "local"
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
	}))

	cfg, err := config.LoadFromDir[*migration.Config](env, "./config/migration")
	if err != nil {
		logger.Error("load config failed", slog.Any("error", err))
		os.Exit(1)
	}

	slog.SetLogLoggerLevel(cfg.Log.Level)
	slog.SetDefault(logger)

	logger = logger.With(slog.String("env", cfg.Env))

	logger.Info("starting migration",
		slog.String("name", cfg.Name),
	)

	err = runMigrate(cfg)
	if err != nil {
		logger.Error("migrate failed", slog.Any("error", err))
		os.Exit(1)
	}

	err = runSeed(cfg)
	if err != nil {
		logger.Error("seed failed", slog.Any("error", err))
		os.Exit(1)
	}

	logger.Info("started")

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM)

	<-interrupt

	logger.Info("stopping...")

	os.Exit(0)
}
