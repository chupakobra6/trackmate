package bot

import (
	"context"
	"strings"
	"time"

	appgoals "github.com/igor/trackmate/internal/app/goals"
	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/storage/postgres"
	"github.com/igor/trackmate/internal/telegram"
	"github.com/igor/trackmate/internal/ui"
)

func (s *Service) handleGoalsConfigure(ctx context.Context, callback telegram.CallbackQuery) (CallbackAnswer, error) {
	workspace, err := s.ensureWorkspaceLoaded(ctx, callback.Message.Chat.ID)
	if err != nil || workspace.ID == 0 {
		return CallbackAnswer{Text: "Не получилось найти настройки группы."}, err
	}
	var answer CallbackAnswer
	err = s.Store.InTx(ctx, func(q *postgres.Queries) error {
		if pending, found, err := q.GetPendingInput(ctx, workspace.ID, callback.From.ID); err != nil {
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
			Text:                ui.SeasonalGoalsPrompt(),
			DisableNotification: true,
		})
		if err != nil {
			return err
		}
		_, err = q.UpsertPendingInput(ctx, workspace.ID, callback.From.ID, domain.PendingSeasonalGoals, map[string]any{
			"thread_id":         callback.Message.MessageThreadID,
			"prompt_message_id": prompt.MessageID,
		})
		return err
	})
	return answer, err
}

func (s *Service) consumeSeasonalGoals(ctx context.Context, workspace postgres.Workspace, message telegram.Message, pending postgres.PendingInput) error {
	if strings.TrimSpace(messagePlainText(message)) == "" {
		_ = s.editMessageSafe(ctx, message.Chat.ID, payloadInt64(pending.Payload, "prompt_message_id"), "⚠️ <b>Пришли цели текстом.</b>\n\n"+ui.SeasonalGoalsPrompt(), nil)
		return nil
	}
	return s.Store.InTx(ctx, func(q *postgres.Queries) error {
		if _, ok, err := q.ClaimPendingInput(ctx, workspace.ID, message.From.ID, domain.PendingSeasonalGoals); err != nil || !ok {
			return err
		}
		participant, err := q.RegisterParticipant(ctx, workspace.ID, message.From.ID, message.From.Username, telegram.DisplayName(*message.From))
		if err != nil {
			return err
		}
		now, err := q.CurrentNow(ctx, time.Now().UTC())
		if err != nil {
			return err
		}
		period, err := domain.CurrentGoalPeriod(workspace.Timezone, now)
		if err != nil {
			return err
		}
		input := telegram.NewMessageInput(message)
		goalSet, err := q.UpsertSeasonalGoalSet(ctx, workspace.ID, participant.ID, message.From.ID, period, input.TextHTML)
		if err != nil {
			return err
		}
		cardText := ui.FormatSeasonalGoalCard(goalSet, telegram.DisplayName(*message.From), message.From.Username, "")
		if goalSet.CardMessageID != nil {
			_ = s.Telegram.EditMessageText(ctx, telegram.EditMessageTextRequest{ChatID: message.Chat.ID, MessageID: *goalSet.CardMessageID, Text: cardText})
		} else {
			card, err := s.Telegram.SendMessage(ctx, telegram.SendMessageRequest{
				ChatID:              message.Chat.ID,
				MessageThreadID:     message.MessageThreadID,
				Text:                cardText,
				DisableNotification: true,
			})
			if err != nil {
				return err
			}
			if err := q.SetSeasonalGoalCardMessageID(ctx, goalSet.ID, card.MessageID, message.MessageThreadID); err != nil {
				return err
			}
		}
		text := "✅ <b>Цели сохранены.</b>\nРаз в неделю я буду просить короткий review в этой теме."
		if !s.editMessageSafe(ctx, message.Chat.ID, payloadInt64(pending.Payload, "prompt_message_id"), text, nil) {
			_, _ = s.Telegram.SendMessage(ctx, telegram.SendMessageRequest{ChatID: message.Chat.ID, MessageThreadID: message.MessageThreadID, Text: text, DisableNotification: true})
		}
		return nil
	})
}

func (s *Service) consumeGoalWeeklyReview(ctx context.Context, workspace postgres.Workspace, message telegram.Message) error {
	return s.Store.InTx(ctx, func(q *postgres.Queries) error {
		pending, ok, err := q.ClaimPendingInput(ctx, workspace.ID, message.From.ID, domain.PendingGoalWeeklyReview)
		if err != nil || !ok {
			return err
		}
		reviewID := payloadInt64(pending.Payload, "review_id")
		input := telegram.NewMessageInput(message)
		review, saved, err := q.SubmitGoalWeeklyReview(ctx, reviewID, message.From.ID, input.TextHTML)
		if err != nil || !saved {
			return err
		}
		_ = s.Telegram.EditMessageText(ctx, telegram.EditMessageTextRequest{
			ChatID:    message.Chat.ID,
			MessageID: payloadInt64(pending.Payload, "prompt_message_id"),
			Text:      ui.FormatGoalWeeklyReviewSaved(review),
		})
		return nil
	})
}

func (s *Service) handleGoalFinalStatus(ctx context.Context, callback telegram.CallbackQuery, goalSetID int64, status domain.GoalFinalStatus) (CallbackAnswer, error) {
	workspace, err := s.ensureWorkspaceLoaded(ctx, callback.Message.Chat.ID)
	if err != nil || workspace.ID == 0 {
		return CallbackAnswer{Text: "Не получилось найти настройки группы."}, err
	}
	var answer CallbackAnswer
	err = s.Store.InTx(ctx, func(q *postgres.Queries) error {
		goalSet, found, err := q.GetSeasonalGoalSet(ctx, goalSetID)
		if err != nil {
			return err
		}
		if !found {
			answer.Text = "Цели не найдены."
			return nil
		}
		if goalSet.OwnerUserID != callback.From.ID {
			answer.Text = "Финальный review может оставить только автор целей."
			return nil
		}
		if pending, found, err := q.GetPendingInput(ctx, workspace.ID, callback.From.ID); err != nil {
			return err
		} else if found {
			answer.Text = pendingBusyText(pending.Kind)
			return nil
		}
		if _, err := q.GetOrCreateGoalFinalReview(ctx, goalSetID); err != nil {
			return err
		}
		review, saved, err := q.SetGoalFinalReviewStatus(ctx, goalSetID, callback.From.ID, status)
		if err != nil || !saved {
			return err
		}
		_, err = q.UpsertPendingInput(ctx, workspace.ID, callback.From.ID, domain.PendingGoalFinalReflection, map[string]any{
			"goal_set_id":       goalSetID,
			"prompt_message_id": callback.Message.MessageID,
			"thread_id":         callback.Message.MessageThreadID,
		})
		if err != nil {
			return err
		}
		selected := status
		if review.Status != nil {
			selected = *review.Status
		}
		_ = s.Telegram.EditMessageText(ctx, telegram.EditMessageTextRequest{
			ChatID:    callback.Message.Chat.ID,
			MessageID: callback.Message.MessageID,
			Text:      ui.FormatGoalFinalReflectionPrompt(goalSet, selected),
		})
		return nil
	})
	return answer, err
}

func (s *Service) consumeGoalFinalReflection(ctx context.Context, workspace postgres.Workspace, message telegram.Message) error {
	return s.Store.InTx(ctx, func(q *postgres.Queries) error {
		pending, ok, err := q.ClaimPendingInput(ctx, workspace.ID, message.From.ID, domain.PendingGoalFinalReflection)
		if err != nil || !ok {
			return err
		}
		goalSetID := payloadInt64(pending.Payload, "goal_set_id")
		goalSet, found, err := q.GetSeasonalGoalSet(ctx, goalSetID)
		if err != nil || !found {
			return err
		}
		input := telegram.NewMessageInput(message)
		review, saved, err := q.CompleteGoalFinalReview(ctx, goalSetID, message.From.ID, input.TextHTML)
		if err != nil || !saved {
			return err
		}
		_ = s.Telegram.EditMessageText(ctx, telegram.EditMessageTextRequest{
			ChatID:    message.Chat.ID,
			MessageID: payloadInt64(pending.Payload, "prompt_message_id"),
			Text:      ui.FormatGoalFinalReviewSaved(goalSet, review),
		})
		return nil
	})
}

func (s *Service) goalNudge(ctx context.Context, q *postgres.Queries, workspace postgres.Workspace, participant postgres.Participant, seed string, status string) (string, error) {
	return appgoals.MaybeNudge(ctx, q, workspace, participant, seed, status, time.Now().UTC())
}
