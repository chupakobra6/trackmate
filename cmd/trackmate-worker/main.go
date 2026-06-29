package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/igor/trackmate/internal/config"
	"github.com/igor/trackmate/internal/control"
	"github.com/igor/trackmate/internal/logging"
	"github.com/igor/trackmate/internal/storage/postgres"
	"github.com/igor/trackmate/internal/telegram"
	"github.com/igor/trackmate/internal/worker"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := cfg.RequireBotToken(); err != nil {
		return err
	}
	logger := logging.New(cfg.LogLevel)
	store, err := postgres.Open(ctx, cfg.DatabaseURL, logger)
	if err != nil {
		return err
	}
	defer store.Close()
	tg := telegram.NewClient(cfg.BotToken, logger)
	runner := &worker.Runner{Store: store, TG: tg, Logger: logger}
	if cfg.ControlEnabled() {
		server := &control.Server{Store: store, Worker: runner, Logger: logger}
		go func() {
			if err := server.ListenAndServe(ctx, cfg.ControlHTTPAddr); err != nil {
				logger.ErrorContext(ctx, "control_server_failed", "error", err)
			}
		}()
	}
	ticker := time.NewTicker(time.Duration(cfg.WorkerTickSeconds) * time.Second)
	defer ticker.Stop()
	for {
		if err := runner.Tick(ctx, time.Now().UTC()); err != nil && !errors.Is(err, worker.ErrWorkerLockBusy) {
			logger.ErrorContext(ctx, "worker_tick_failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
