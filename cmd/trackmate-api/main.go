package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/igor/trackmate/internal/bot"
	"github.com/igor/trackmate/internal/config"
	"github.com/igor/trackmate/internal/dispatcher"
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
	updateDispatcher := dispatcher.New(32, 5*time.Minute)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = updateDispatcher.Shutdown(shutdownCtx)
	}()

	var offset int64
	var pollErrorStreak int
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
		for _, update := range updates {
			offset = update.UpdateID + 1
			update := update
			updateCtx := observability.WithUpdateID(observability.EnsureTraceID(ctx), update.UpdateID)
			telegram.LogIncomingUpdate(updateCtx, logger, update)
			key := updateMailboxKey(update)
			if err := updateDispatcher.Submit(updateCtx, key, func(jobCtx context.Context) {
				answer, err := service.HandleUpdate(jobCtx, update)
				if update.Callback != nil {
					answerID := update.Callback.ID
					text := answer.Text
					if answer.ID != "" {
						answerID = answer.ID
					}
					if err := tg.AnswerCallbackQuery(jobCtx, telegram.AnswerCallbackQueryRequest{CallbackQueryID: answerID, Text: text}); err != nil {
						logger.WarnContext(jobCtx, "answer_callback_failed", "error", err)
					}
				}
				if err != nil {
					logger.ErrorContext(jobCtx, "handle_update_failed", "update_id", update.UpdateID, "error", err)
				}
			}); err != nil {
				if errors.Is(err, dispatcher.ErrDispatcherClosed) || errors.Is(err, context.Canceled) {
					return nil
				}
				logger.ErrorContext(updateCtx, "dispatch_update_failed", "update_id", update.UpdateID, "error", err)
			}
		}
	}
}

func updateMailboxKey(update telegram.Update) string {
	if update.MyChatMember != nil {
		return fmt.Sprintf("chat:%d:setup", update.MyChatMember.Chat.ID)
	}
	if update.Callback != nil {
		if update.Callback.Message != nil {
			if update.Callback.Data == "setup:check" || update.Callback.Data == "setup:start" {
				return fmt.Sprintf("chat:%d:setup", update.Callback.Message.Chat.ID)
			}
			return fmt.Sprintf("chat:%d:user:%d", update.Callback.Message.Chat.ID, update.Callback.From.ID)
		}
		return "user:" + strconv.FormatInt(update.Callback.From.ID, 10)
	}
	if update.Message != nil {
		if update.Message.From != nil {
			if isCommand(update.Message.Text, "/setup") {
				return fmt.Sprintf("chat:%d:setup", update.Message.Chat.ID)
			}
			return fmt.Sprintf("chat:%d:user:%d", update.Message.Chat.ID, update.Message.From.ID)
		}
		return fmt.Sprintf("chat:%d", update.Message.Chat.ID)
	}
	return fmt.Sprintf("update:%d", update.UpdateID)
}

func isCommand(text string, command string) bool {
	if len(text) < len(command) {
		return false
	}
	token := text
	for i, r := range text {
		if r == ' ' || r == '\n' || r == '\t' {
			token = text[:i]
			break
		}
	}
	for i, r := range token {
		if r == '@' {
			token = token[:i]
			break
		}
	}
	return token == command
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
