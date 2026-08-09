package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/babafemi99/daftar/backend/internal/cfg"
	"github.com/babafemi99/daftar/backend/internal/observability"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(runHealthcheck())
	}
	flags := flag.NewFlagSet("daftar-api", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	seed := flags.Bool("seed", false, "create idempotent reviewer demo data and exit")
	if err := flags.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(os.Stderr, "unexpected arguments: %v\n", flags.Args())
		os.Exit(2)
	}

	config, err := cfg.Load()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	logger := observability.NewLogger(os.Stdout, config)
	slog.SetDefault(logger)
	logger.Info("application starting", "address", config.HTTP.Address)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	application, err := setup(ctx, config)
	if err != nil {
		slog.Error("application startup failed", "error", err)
		os.Exit(1)
	}
	if *seed {
		if err := seedDemo(ctx, application.Dependencies, config.BootstrapUser); err != nil {
			logger.Error("demo seed failed", "error", err)
			_ = application.Shutdown(context.Background())
			os.Exit(1)
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), config.HTTP.ShutdownTimeout)
		defer cancel()
		if err := application.Shutdown(shutdownCtx); err != nil {
			logger.Error("application shutdown failed", "error", err)
			os.Exit(1)
		}
		logger.Info("demo seed complete")
		return
	}
	logger.Info("application ready", "address", config.HTTP.Address)
	printBanner(config)

	serverErr := make(chan error, 1)
	go func() { serverErr <- application.API.Serve() }()

	select {
	case err := <-serverErr:
		if err != nil {
			slog.Error("HTTP server failed", "error", err)
		}
	case <-ctx.Done():
		logger.Info("shutdown requested", "reason", ctx.Err())
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), config.HTTP.ShutdownTimeout)
	defer cancel()
	if err := application.Shutdown(shutdownCtx); err != nil {
		slog.Error("application shutdown failed", "error", err)
	}
	logger.Info("application stopped")
}
