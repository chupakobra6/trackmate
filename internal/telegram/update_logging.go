package telegram

import (
	"context"
	"log/slog"

	"github.com/igor/trackmate/internal/observability"
)

func LogIncomingUpdate(ctx context.Context, logger *slog.Logger, update Update) {
	if logger == nil {
		return
	}
	switch {
	case update.Message != nil:
		msg := update.Message
		var userID int64
		var username string
		if msg.From != nil {
			userID = msg.From.ID
			username = msg.From.Username
		}
		logger.InfoContext(ctx, "telegram_incoming_message", observability.LogAttrs(ctx,
			"chat_id", msg.Chat.ID,
			"message_id", msg.MessageID,
			"thread_id", msg.MessageThreadID,
			"user_id", userID,
			"username", username,
			"media_group_id", msg.MediaGroupID,
		)...)
	case update.EditedMessage != nil:
		msg := update.EditedMessage
		var userID int64
		var username string
		if msg.From != nil {
			userID = msg.From.ID
			username = msg.From.Username
		}
		logger.InfoContext(ctx, "telegram_incoming_edited_message", observability.LogAttrs(ctx,
			"chat_id", msg.Chat.ID,
			"message_id", msg.MessageID,
			"thread_id", msg.MessageThreadID,
			"user_id", userID,
			"username", username,
			"media_group_id", msg.MediaGroupID,
		)...)
	case update.Callback != nil:
		cb := update.Callback
		var chatID int64
		var messageID int64
		var threadID int64
		if cb.Message != nil {
			chatID = cb.Message.Chat.ID
			messageID = cb.Message.MessageID
			threadID = cb.Message.MessageThreadID
		}
		logger.InfoContext(ctx, "telegram_incoming_callback", observability.LogAttrs(ctx,
			"chat_id", chatID,
			"message_id", messageID,
			"thread_id", threadID,
			"user_id", cb.From.ID,
			"username", cb.From.Username,
			"data", cb.Data,
		)...)
	}
}
