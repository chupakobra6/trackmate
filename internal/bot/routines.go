package bot

import (
	"context"
	"strings"
	"time"

	approutine "github.com/igor/trackmate/internal/app/routine"
	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/messages"
	"github.com/igor/trackmate/internal/storage/postgres"
	"github.com/igor/trackmate/internal/telegram"
	"github.com/igor/trackmate/internal/ui"
)

func (s *Service) handleRoutineConfigure(ctx context.Context, callback telegram.CallbackQuery) (CallbackAnswer, error) {
	workspace, err := s.ensureWorkspaceLoaded(ctx, callback.Message.Chat.ID)
	if err != nil || workspace.ID == 0 {
		return CallbackAnswer{Text: messages.Text("callback.workspace_missing")}, err
	}
	var answer CallbackAnswer
	err = s.Store.InTx(ctx, func(q *postgres.Queries) error {
		if pending, found, err := q.GetPendingInput(ctx, workspace.ID, callback.From.ID, callback.Message.MessageThreadID); err != nil {
			return err
		} else if found {
			answer.Text = pendingBusyText(pending.Kind)
			return nil
		}
		if _, err := q.RegisterParticipant(ctx, workspace.ID, callback.From.ID, callback.From.Username, telegram.DisplayName(callback.From)); err != nil {
			return err
		}
		prompt, err := s.Telegram.SendMessage(ctx, telegram.SendMessageRequest{
			ChatID:              callback.Message.Chat.ID,
			MessageThreadID:     callback.Message.MessageThreadID,
			Text:                ui.RoutinePlanPrompt(),
			DisableNotification: true,
		})
		if err != nil {
			return err
		}
		_, err = q.UpsertPendingInput(ctx, workspace.ID, callback.From.ID, callback.Message.MessageThreadID, domain.PendingRoutinePlan, map[string]any{
			"thread_id":         callback.Message.MessageThreadID,
			"prompt_message_id": prompt.MessageID,
		})
		return err
	})
	return answer, err
}

func (s *Service) consumeRoutinePlan(ctx context.Context, workspace postgres.Workspace, message telegram.Message, pending postgres.PendingInput) error {
	raw := messagePlainText(message)
	items, parseErr := domain.ParseRoutineItems(raw)
	if raw == "" || parseErr != nil {
		text := messages.Text("routine.plan.invalid")
		if parseErr != nil && strings.Contains(parseErr.Error(), "max") {
			text = messages.Text("routine.plan.too_many")
		}
		_ = s.refreshPendingInputActivity(ctx, workspace.ID, message.From.ID, message.MessageThreadID, pending, message.MessageID)
		_ = s.editMessageSafe(ctx, message.Chat.ID, payloadInt64(pending.Payload, "prompt_message_id"), text+"\n\n"+ui.RoutinePlanPrompt(), nil)
		return nil
	}
	return s.Store.InTx(ctx, func(q *postgres.Queries) error {
		claimed, ok, err := q.ClaimPendingInput(ctx, workspace.ID, message.From.ID, message.MessageThreadID, domain.PendingRoutinePlan)
		if err != nil || !ok {
			return err
		}
		participant, err := q.RegisterParticipant(ctx, workspace.ID, message.From.ID, message.From.Username, telegram.DisplayName(*message.From))
		if err != nil {
			return err
		}
		if _, err := q.UpsertRoutinePlan(ctx, workspace.ID, participant.ID, message.From.ID, items); err != nil {
			return err
		}
		s.deletePendingUserMessages(ctx, message.Chat.ID, claimed.Payload)
		_ = s.Telegram.DeleteMessage(ctx, message.Chat.ID, message.MessageID)
		_ = s.Telegram.DeleteMessage(ctx, message.Chat.ID, payloadInt64(claimed.Payload, "prompt_message_id"))
		return nil
	})
}

func (s *Service) handleRoutineItem(ctx context.Context, callback telegram.CallbackQuery, checkinID int64, itemIndex int, status domain.RoutineItemStatus) (CallbackAnswer, error) {
	workspace, err := s.ensureWorkspaceLoaded(ctx, callback.Message.Chat.ID)
	if err != nil || workspace.ID == 0 {
		return CallbackAnswer{Text: messages.Text("callback.workspace_missing")}, err
	}
	var answer CallbackAnswer
	err = s.Store.InTx(ctx, func(q *postgres.Queries) error {
		checkin, found, err := q.GetRoutineCheckin(ctx, checkinID)
		if err != nil {
			return err
		}
		if !found {
			answer.Text = messages.Text("routine.checkin.not_found")
			return nil
		}
		if checkin.OwnerUserID != callback.From.ID {
			answer.Text = messages.Text("routine.checkin.author_only")
			return nil
		}
		if checkin.CompletedAt != nil {
			answer.Text = messages.Text("routine.checkin.completed")
			return nil
		}
		nextIndex := ui.NextRoutineItemIndex(checkin)
		if nextIndex != itemIndex {
			answer.Text = messages.Text("routine.checkin.stale_item")
			return nil
		}
		if pending, found, err := q.GetPendingInput(ctx, workspace.ID, callback.From.ID, callback.Message.MessageThreadID); err != nil {
			return err
		} else if found {
			answer.Text = pendingBusyText(pending.Kind)
			return nil
		}
		if status == domain.RoutineItemPartial || status == domain.RoutineItemFailed {
			updated, ok, err := q.SetRoutineCheckinItemStatus(ctx, checkinID, callback.From.ID, itemIndex, status, nil)
			if err != nil || !ok {
				return err
			}
			_ = s.Telegram.EditMessageText(ctx, telegram.EditMessageTextRequest{
				ChatID:      callback.Message.Chat.ID,
				MessageID:   callback.Message.MessageID,
				Text:        ui.FormatRoutineCheckinStatusCard(updated, telegram.DisplayName(callback.From), callback.From.Username, ""),
				ReplyMarkup: ui.EmptyKeyboard(),
			})
			prompt, err := s.Telegram.SendMessage(ctx, telegram.SendMessageRequest{
				ChatID:              callback.Message.Chat.ID,
				MessageThreadID:     callback.Message.MessageThreadID,
				Text:                ui.FormatRoutineReasonPrompt(checkin.Items[itemIndex].Text),
				ReplyToMessageID:    callback.Message.MessageID,
				DisableNotification: true,
			})
			if err != nil {
				return err
			}
			_, err = q.UpsertPendingInput(ctx, workspace.ID, callback.From.ID, callback.Message.MessageThreadID, domain.PendingRoutineReason, map[string]any{
				"checkin_id":        checkinID,
				"item_index":        itemIndex,
				"status":            string(status),
				"card_message_id":   callback.Message.MessageID,
				"prompt_message_id": prompt.MessageID,
				"thread_id":         callback.Message.MessageThreadID,
			})
			return err
		}
		updated, ok, err := q.SetRoutineCheckinItemStatus(ctx, checkinID, callback.From.ID, itemIndex, status, nil)
		if err != nil || !ok {
			return err
		}
		return s.advanceRoutineCheckin(ctx, q, workspace, callback.Message.Chat.ID, callback.Message.MessageID, callback.Message.MessageThreadID, callback.From, updated)
	})
	return answer, err
}

func (s *Service) consumeRoutineReason(ctx context.Context, workspace postgres.Workspace, message telegram.Message) error {
	return s.Store.InTx(ctx, func(q *postgres.Queries) error {
		pending, ok, err := q.ClaimPendingInput(ctx, workspace.ID, message.From.ID, message.MessageThreadID, domain.PendingRoutineReason)
		if err != nil || !ok {
			return err
		}
		checkinID := payloadInt64(pending.Payload, "checkin_id")
		itemIndex := int(payloadInt64(pending.Payload, "item_index"))
		status := domain.RoutineItemStatus(payloadString(pending.Payload, "status"))
		input := telegram.NewMessageInput(message)
		reason := input.TextHTML
		updated, saved, err := q.SetRoutineCheckinItemStatus(ctx, checkinID, message.From.ID, itemIndex, status, &reason)
		if err != nil || !saved {
			return err
		}
		_ = s.Telegram.DeleteMessage(ctx, message.Chat.ID, payloadInt64(pending.Payload, "prompt_message_id"))
		_ = s.Telegram.DeleteMessage(ctx, message.Chat.ID, message.MessageID)
		cardMessageID := payloadInt64(pending.Payload, "card_message_id")
		if cardMessageID == 0 {
			cardMessageID = payloadInt64(pending.Payload, "prompt_message_id")
		}
		return s.advanceRoutineCheckin(ctx, q, workspace, message.Chat.ID, cardMessageID, message.MessageThreadID, *message.From, updated)
	})
}

func (s *Service) advanceRoutineCheckin(ctx context.Context, q *postgres.Queries, workspace postgres.Workspace, chatID int64, messageID int64, threadID int64, user telegram.User, checkin postgres.RoutineCheckin) error {
	nextIndex := ui.NextRoutineItemIndex(checkin)
	if nextIndex >= 0 {
		return s.Telegram.EditMessageText(ctx, telegram.EditMessageTextRequest{
			ChatID:      chatID,
			MessageID:   messageID,
			Text:        ui.FormatRoutineCheckinCard(checkin, telegram.DisplayName(user), user.Username, ""),
			ReplyMarkup: ui.RoutineItemKeyboard(checkin.ID, nextIndex),
		})
	}
	completed, ok, err := q.CompleteRoutineCheckinWithoutReflection(ctx, checkin.ID, user.ID)
	if err != nil || !ok {
		return err
	}
	if completed.ReminderMessageID != nil {
		_ = s.Telegram.DeleteMessage(ctx, chatID, *completed.ReminderMessageID)
		if err := q.ClearRoutineCheckinReminderMessageID(ctx, completed.ID); err != nil {
			return err
		}
	}
	_ = s.Telegram.DeleteMessage(ctx, chatID, messageID)
	now, err := q.CurrentNow(ctx, time.Now().UTC())
	if err != nil {
		return err
	}
	return approutine.RefreshLeaderboard(ctx, q, s.Telegram, workspace, chatID, now)
}
