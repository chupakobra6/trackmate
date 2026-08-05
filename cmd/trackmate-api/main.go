package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/igor/trackmate/internal/bot"
	"github.com/igor/trackmate/internal/config"
	"github.com/igor/trackmate/internal/logging"
	"github.com/igor/trackmate/internal/observability"
	"github.com/igor/trackmate/internal/storage/postgres"
	"github.com/igor/trackmate/internal/telegram"
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
	me, err := tg.GetMe(ctx)
	if err != nil {
		return err
	}
	service := bot.NewService(store, tg, logger, cfg.DefaultTimezone, me.ID)

	var offset int64
	var pollErrorStreak int
	var updateErrorStreak int
	for {
		updates, err := tg.PollUpdates(ctx, offset, cfg.PollTimeout)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			pollErrorStreak++
			logger.WarnContext(ctx, "poll_updates_failed", "streak", pollErrorStreak, "error", err)
			select {
			case <-time.After(pollRetryDelay(pollErrorStreak)):
			case <-ctx.Done():
				return nil
			}
			continue
		}
		pollErrorStreak = 0
		nextOffset, err := processUpdates(ctx, offset, updates, func(batchCtx context.Context, update telegram.Update) error {
			updateCtx := observability.WithUpdateID(observability.EnsureTraceID(batchCtx), update.UpdateID)
			telegram.LogIncomingUpdate(updateCtx, logger, update)
			answer, handleErr := service.HandleUpdate(updateCtx, update)
			if update.Callback != nil {
				answerID := update.Callback.ID
				text := answer.Text
				if answer.ID != "" {
					answerID = answer.ID
				}
				if err := tg.AnswerCallbackQuery(updateCtx, telegram.AnswerCallbackQueryRequest{CallbackQueryID: answerID, Text: text}); err != nil {
					logger.WarnContext(updateCtx, "answer_callback_failed", "error", err)
				}
			}
			return handleErr
		})
		offset = nextOffset
		if err == nil {
			updateErrorStreak = 0
			continue
		}
		if errors.Is(err, context.Canceled) {
			return nil
		}
		updateErrorStreak++
		logger.ErrorContext(ctx, "handle_update_failed", "next_offset", nextOffset, "streak", updateErrorStreak, "error", err)
		select {
		case <-time.After(pollRetryDelay(updateErrorStreak)):
		case <-ctx.Done():
			return nil
		}
	}
}

func processUpdates(ctx context.Context, offset int64, updates []telegram.Update, handle func(context.Context, telegram.Update) error) (int64, error) {
	nextOffset := offset
	for _, update := range updates {
		if update.UpdateID < nextOffset {
			continue
		}
		if err := handle(ctx, update); err != nil {
			return nextOffset, fmt.Errorf("handle update %d: %w", update.UpdateID, err)
		}
		nextOffset = update.UpdateID + 1
	}
	return nextOffset, nil
}

func pollRetryDelay(streak int) time.Duration {
	switch {
	case streak <= 1:
		return time.Second
	case streak <= 5:
		return 2 * time.Second
	case streak <= 15:
		return 5 * time.Second
	default:
		return 15 * time.Second
	}
}
