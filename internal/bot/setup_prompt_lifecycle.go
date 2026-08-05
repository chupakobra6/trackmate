package bot

import (
	"context"
	"fmt"

	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/storage/postgres"
	"github.com/igor/trackmate/internal/telegram"
)

// openSetupPrompt owns the recoverable lifecycle shared by Routine and Goals
// setup. A repeated callback reuses the stored prompt. A replacement is sent
// only when Telegram explicitly says that the stored message no longer exists.
func (s *Service) openSetupPrompt(ctx context.Context, workspace postgres.Workspace, callback telegram.CallbackQuery, kind domain.PendingInputKind, text string) (CallbackAnswer, error) {
	var answer CallbackAnswer
	err := s.Store.InTx(ctx, func(q *postgres.Queries) error {
		// RegisterParticipant uses INSERT ... ON CONFLICT DO UPDATE. PostgreSQL
		// keeps that participant row locked until this transaction commits, so
		// concurrent configure callbacks for one user cannot both send a prompt.
		if _, err := q.RegisterParticipant(ctx, workspace.ID, callback.From.ID, callback.From.Username, telegram.DisplayName(callback.From)); err != nil {
			return err
		}
		pending, found, err := q.GetPendingInput(ctx, workspace.ID, callback.From.ID, callback.Message.MessageThreadID)
		if err != nil {
			return err
		}
		if found && pending.Kind != kind {
			answer.Text = pendingBusyText(pending.Kind)
			return nil
		}
		if found {
			messageID := payloadInt64(pending.Payload, "prompt_message_id")
			if messageID != 0 {
				err := s.Telegram.EditMessageText(ctx, telegram.EditMessageTextRequest{
					ChatID:    callback.Message.Chat.ID,
					MessageID: messageID,
					Text:      text,
				})
				if err == nil || telegram.IsNotModifiedError(err) {
					answer.Text = pendingBusyText(kind)
					return nil
				}
				if !telegram.IsMissingEditTarget(err) {
					return fmt.Errorf("verify %s setup prompt: %w", kind, err)
				}
				s.warnMessageLifecycle(ctx, "setup_prompt_missing", callback.Message.Chat.ID, messageID, err)
			}
		}

		prompt, err := s.Telegram.SendMessage(ctx, telegram.SendMessageRequest{
			ChatID:              callback.Message.Chat.ID,
			MessageThreadID:     callback.Message.MessageThreadID,
			Text:                text,
			DisableNotification: true,
		})
		if err != nil {
			return err
		}
		if _, err := q.UpsertPendingInput(ctx, workspace.ID, callback.From.ID, callback.Message.MessageThreadID, kind, map[string]any{
			"thread_id":         callback.Message.MessageThreadID,
			"prompt_message_id": prompt.MessageID,
		}); err != nil {
			_ = s.Telegram.DeleteMessage(ctx, callback.Message.Chat.ID, prompt.MessageID)
			return err
		}
		return nil
	})
	return answer, err
}
