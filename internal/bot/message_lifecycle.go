package bot

import (
	"context"
	"errors"
	"fmt"

	"github.com/igor/trackmate/internal/messages"
	"github.com/igor/trackmate/internal/storage/postgres"
	"github.com/igor/trackmate/internal/telegram"
	"github.com/igor/trackmate/internal/ui"
)

type replyMarkupEditor interface {
	EditMessageReplyMarkup(context.Context, telegram.EditMessageReplyMarkupRequest) error
}

// showDailyEntryStatus advances the message that received the callback instead
// of creating a second message. This makes callback replay naturally
// idempotent: every delivery targets the same Telegram message ID.
func (s *Service) showDailyEntryStatus(ctx context.Context, chatID int64, messageID int64, task postgres.DailyTask) error {
	err := s.Telegram.EditMessageText(ctx, telegram.EditMessageTextRequest{
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        messages.Text("task.status.prompt"),
		ReplyMarkup: dailyEntryStatusKeyboard(task),
	})
	if telegram.IsNotModifiedError(err) {
		return nil
	}
	return err
}

// dismissTelegramMessage makes a notice inert even when Telegram refuses to
// delete an older message. A successful return guarantees that the message is
// either gone, replaced with a closed tombstone, or at least has no buttons.
func (s *Service) dismissTelegramMessage(ctx context.Context, chatID int64, messageID int64) error {
	if messageID == 0 {
		return nil
	}

	deleteErr := s.Telegram.DeleteMessage(ctx, chatID, messageID)
	if deleteErr == nil {
		return nil
	}
	s.warnMessageLifecycle(ctx, "notice_delete_failed", chatID, messageID, deleteErr)

	editErr := s.Telegram.EditMessageText(ctx, telegram.EditMessageTextRequest{
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        messages.Text("notice.dismissed"),
		ReplyMarkup: ui.EmptyKeyboard(),
	})
	if editErr == nil || telegram.IsNotModifiedError(editErr) {
		return nil
	}
	s.warnMessageLifecycle(ctx, "notice_tombstone_failed", chatID, messageID, editErr)

	editor, ok := s.Telegram.(replyMarkupEditor)
	if !ok {
		return errors.Join(deleteErr, editErr, errors.New("telegram API cannot remove reply markup"))
	}
	markupErr := editor.EditMessageReplyMarkup(ctx, telegram.EditMessageReplyMarkupRequest{
		ChatID:    chatID,
		MessageID: messageID,
	})
	if markupErr == nil {
		return nil
	}
	s.warnMessageLifecycle(ctx, "notice_keyboard_remove_failed", chatID, messageID, markupErr)
	return errors.Join(deleteErr, editErr, markupErr)
}

func (s *Service) warnMessageLifecycle(ctx context.Context, event string, chatID int64, messageID int64, err error) {
	if s.Logger == nil {
		return
	}
	s.Logger.WarnContext(ctx, event, "chat_id", chatID, "message_id", messageID, "error", fmt.Sprint(err))
}
