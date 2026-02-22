//nolint:sloglint // global logger is acceptable in main function
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"vwap/internal/api"
	"vwap/internal/config"
	apiCfg "vwap/internal/config/api"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
	}))
	slog.SetDefault(logger)

	slog.Info("starting...")

	env := os.Getenv("ENV")
	if env == "" {
		env = "local"
	}

	cfg, err := config.LoadFromDir[*apiCfg.Config](env, "./config/api")
	if err != nil {
		slog.Error("load config failed", slog.Any("error", err))
		os.Exit(1)
	}

	slog.SetLogLoggerLevel(cfg.Log.Level)

	ctx := context.Background()

	apiServer, err := api.NewServer(ctx, cfg.AppConfig)
	if err != nil {
		slog.Error("new server failed", slog.Any("error", err))
		os.Exit(1)
	}

	shutdownFn := apiServer.Start()

	slog.Info("started")

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM)

	<-interrupt

	slog.Info("stopping...")

	err = shutdownFn(ctx)
	if err != nil {
		slog.Error("shutdown failed", slog.Any("error", err))
		os.Exit(1)
	}
}
