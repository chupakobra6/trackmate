package messagecleanup

import (
	"context"
	"time"

	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/telegram"
)

type API interface {
	DeleteMessage(context.Context, int64, int64) error
	EditMessageText(context.Context, telegram.EditMessageTextRequest) error
}

// DeleteBestEffort is for scheduled cleanup of messages that do not have an
// editable fallback, such as user input. It never calls deleteMessage outside
// Telegram's documented deletion window and never blocks the owning worker.
func DeleteBestEffort(ctx context.Context, tg API, chatID int64, messageID int64, sentAt time.Time, nowUTC time.Time) {
	if messageID == 0 || !domain.TelegramMessageDeleteAllowed(sentAt, nowUTC) {
		return
	}
	_ = tg.DeleteMessage(ctx, chatID, messageID)
}

// CloseBestEffort deletes a scheduled bot message while Telegram still allows
// it. If the window expired or deletion fails, it applies the supplied inert
// edit instead. Neither outcome can block the owning worker lifecycle.
func CloseBestEffort(ctx context.Context, tg API, chatID int64, messageID int64, sentAt time.Time, nowUTC time.Time, fallback telegram.EditMessageTextRequest) {
	if messageID == 0 {
		return
	}
	if domain.TelegramMessageDeleteAllowed(sentAt, nowUTC) {
		if err := tg.DeleteMessage(ctx, chatID, messageID); err == nil {
			return
		}
	}
	fallback.ChatID = chatID
	fallback.MessageID = messageID
	_ = tg.EditMessageText(ctx, fallback)
}
