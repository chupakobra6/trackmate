package bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	approutine "github.com/igor/trackmate/internal/app/routine"
	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/storage/postgres"
	"github.com/igor/trackmate/internal/telegram"
	"github.com/igor/trackmate/internal/ui"
)

func (s *Service) handleRoutineConfigure(ctx context.Context, callback telegram.CallbackQuery) (CallbackAnswer, error) {
	workspace, err := s.ensureWorkspaceLoaded(ctx, callback.Message.Chat.ID)
	if err != nil || workspace.ID == 0 {
		return CallbackAnswer{Text: "Не получилось найти настройки группы"}, err
	}
	var answer CallbackAnswer
	err = s.Store.InTx(ctx, func(q *postgres.Queries) error {
		if pending, found, err := q.GetPendingInput(ctx, workspace.ID, callback.From.ID); err != nil {
			return err
		} else if found {
			cancelled, err := s.cancelSwitchableSetupInput(ctx, q, callback.Message.Chat.ID, pending)
			if err != nil {
				return err
			}
			if cancelled {
				answer.Text = "Предыдущий ввод сброшен"
			} else {
				answer.Text = pendingBusyText(pending.Kind)
				return nil
			}
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
		_, err = q.UpsertPendingInput(ctx, workspace.ID, callback.From.ID, domain.PendingRoutinePlan, map[string]any{
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
		text := "⚠️ <b>Пришли список текстом: один пункт на строку</b>"
		if parseErr != nil && strings.Contains(parseErr.Error(), "max") {
			text = "⚠️ <b>Слишком много пунктов</b>\nОграничение — не более 9 рутин"
		}
		_ = s.editMessageSafe(ctx, message.Chat.ID, payloadInt64(pending.Payload, "prompt_message_id"), text+"\n\n"+ui.RoutinePlanPrompt(), nil)
		return nil
	}
	return s.Store.InTx(ctx, func(q *postgres.Queries) error {
		if _, ok, err := q.ClaimPendingInput(ctx, workspace.ID, message.From.ID, domain.PendingRoutinePlan); err != nil || !ok {
			return err
		}
		participant, err := q.RegisterParticipant(ctx, workspace.ID, message.From.ID, message.From.Username, telegram.DisplayName(*message.From))
		if err != nil {
			return err
		}
		if _, err := q.UpsertRoutinePlan(ctx, workspace.ID, participant.ID, message.From.ID, items); err != nil {
			return err
		}
		text := fmt.Sprintf("✅ <b>Рутины сохранены</b>\nВсего пунктов: %d\nС завтрашнего дня после 09:00 буду присылать карточку для отметок", len(items))
		if !s.editMessageSafe(ctx, message.Chat.ID, payloadInt64(pending.Payload, "prompt_message_id"), text, nil) {
			_, _ = s.Telegram.SendMessage(ctx, telegram.SendMessageRequest{ChatID: message.Chat.ID, MessageThreadID: message.MessageThreadID, Text: text, DisableNotification: true})
		}
		return nil
	})
}

func (s *Service) handleRoutineItem(ctx context.Context, callback telegram.CallbackQuery, checkinID int64, itemIndex int, status domain.RoutineItemStatus) (CallbackAnswer, error) {
	workspace, err := s.ensureWorkspaceLoaded(ctx, callback.Message.Chat.ID)
	if err != nil || workspace.ID == 0 {
		return CallbackAnswer{Text: "Не получилось найти настройки группы"}, err
	}
	var answer CallbackAnswer
	err = s.Store.InTx(ctx, func(q *postgres.Queries) error {
		checkin, found, err := q.GetRoutineCheckin(ctx, checkinID)
		if err != nil {
			return err
		}
		if !found {
			answer.Text = "Проверка не найдена"
			return nil
		}
		if checkin.OwnerUserID != callback.From.ID {
			answer.Text = "Отметить рутину может только ее автор"
			return nil
		}
		if checkin.CompletedAt != nil {
			answer.Text = "Эта проверка уже завершена"
			return nil
		}
		nextIndex := ui.NextRoutineItemIndex(checkin)
		if nextIndex != itemIndex {
			answer.Text = "Этот пункт уже не актуален"
			return nil
		}
		if pending, found, err := q.GetPendingInput(ctx, workspace.ID, callback.From.ID); err != nil {
			return err
		} else if found {
			answer.Text = pendingBusyText(pending.Kind)
			return nil
		}
		if status == domain.RoutineItemPartial || status == domain.RoutineItemFailed {
			_, err := q.UpsertPendingInput(ctx, workspace.ID, callback.From.ID, domain.PendingRoutineReason, map[string]any{
				"checkin_id":        checkinID,
				"item_index":        itemIndex,
				"status":            string(status),
				"prompt_message_id": callback.Message.MessageID,
				"thread_id":         callback.Message.MessageThreadID,
			})
			if err != nil {
				return err
			}
			_ = s.Telegram.EditMessageText(ctx, telegram.EditMessageTextRequest{
				ChatID:    callback.Message.Chat.ID,
				MessageID: callback.Message.MessageID,
				Text:      ui.FormatRoutineReasonPrompt(checkin, itemIndex),
			})
			return nil
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
		pending, ok, err := q.ClaimPendingInput(ctx, workspace.ID, message.From.ID, domain.PendingRoutineReason)
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
		return s.advanceRoutineCheckin(ctx, q, workspace, message.Chat.ID, payloadInt64(pending.Payload, "prompt_message_id"), message.MessageThreadID, *message.From, updated)
	})
}

func (s *Service) consumeRoutineReflection(ctx context.Context, workspace postgres.Workspace, message telegram.Message) error {
	return s.Store.InTx(ctx, func(q *postgres.Queries) error {
		pending, ok, err := q.ClaimPendingInput(ctx, workspace.ID, message.From.ID, domain.PendingRoutineReflection)
		if err != nil || !ok {
			return err
		}
		checkinID := payloadInt64(pending.Payload, "checkin_id")
		input := telegram.NewMessageInput(message)
		checkin, completed, err := q.CompleteRoutineCheckin(ctx, checkinID, message.From.ID, input.TextHTML)
		if err != nil || !completed {
			return err
		}
		_ = s.Telegram.EditMessageText(ctx, telegram.EditMessageTextRequest{
			ChatID:    message.Chat.ID,
			MessageID: payloadInt64(pending.Payload, "prompt_message_id"),
			Text:      ui.FormatRoutineCheckinCard(checkin, telegram.DisplayName(*message.From), message.From.Username, ""),
		})
		now, err := q.CurrentNow(ctx, time.Now().UTC())
		if err != nil {
			return err
		}
		return approutine.RefreshLeaderboard(ctx, q, s.Telegram, workspace, message.Chat.ID, now)
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
	if pending, found, err := q.GetPendingInput(ctx, workspace.ID, user.ID); err != nil {
		return err
	} else if found && pending.Kind != domain.PendingRoutineReflection {
		return nil
	}
	if _, err := q.UpsertPendingInput(ctx, workspace.ID, user.ID, domain.PendingRoutineReflection, map[string]any{
		"checkin_id":        checkin.ID,
		"prompt_message_id": messageID,
		"thread_id":         threadID,
	}); err != nil {
		return err
	}
	return s.Telegram.EditMessageText(ctx, telegram.EditMessageTextRequest{
		ChatID:    chatID,
		MessageID: messageID,
		Text:      ui.FormatRoutineReflectionPrompt(checkin, telegram.DisplayName(user), user.Username),
	})
}
