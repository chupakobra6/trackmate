package bot

import (
	"context"
	"errors"
	"strings"
	"time"

	appgoals "github.com/igor/trackmate/internal/app/goals"
	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/messages"
	"github.com/igor/trackmate/internal/storage/postgres"
	"github.com/igor/trackmate/internal/telegram"
	"github.com/igor/trackmate/internal/ui"
)

var errGoalInputNotAccepted = errors.New("goal input was not accepted")

func (s *Service) handleGoalsConfigure(ctx context.Context, callback telegram.CallbackQuery) (CallbackAnswer, error) {
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
			Text:                ui.SeasonalGoalsPrompt(),
			DisableNotification: true,
		})
		if err != nil {
			return err
		}
		_, err = q.UpsertPendingInput(ctx, workspace.ID, callback.From.ID, callback.Message.MessageThreadID, domain.PendingSeasonalGoals, map[string]any{
			"thread_id":         callback.Message.MessageThreadID,
			"prompt_message_id": prompt.MessageID,
		})
		return err
	})
	return answer, err
}

func (s *Service) consumeSeasonalGoals(ctx context.Context, workspace postgres.Workspace, message telegram.Message, pending postgres.PendingInput) error {
	if strings.TrimSpace(messagePlainText(message)) == "" {
		_ = s.refreshPendingInputActivity(ctx, workspace.ID, message.From.ID, message.MessageThreadID, pending, message.MessageID)
		_ = s.editMessageSafe(ctx, message.Chat.ID, payloadInt64(pending.Payload, "prompt_message_id"), messages.Text("goals.invalid")+"\n\n"+ui.SeasonalGoalsPrompt(), nil)
		return nil
	}
	return s.Store.InTx(ctx, func(q *postgres.Queries) error {
		claimed, ok, err := q.ClaimPendingInput(ctx, workspace.ID, message.From.ID, message.MessageThreadID, domain.PendingSeasonalGoals)
		if err != nil || !ok {
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
		sourceMessageID := message.MessageID
		sourceThreadID := message.MessageThreadID
		goalSet, err := q.UpsertSeasonalGoalSet(ctx, workspace.ID, participant.ID, message.From.ID, period, input.TextHTML, &sourceMessageID, &sourceThreadID)
		if err != nil {
			return err
		}
		if goalSet.CardMessageID != nil {
			_ = s.Telegram.DeleteMessage(ctx, message.Chat.ID, *goalSet.CardMessageID)
			if err := q.ClearSeasonalGoalCardMessageID(ctx, goalSet.ID); err != nil {
				return err
			}
		}
		s.deletePendingUserMessages(ctx, message.Chat.ID, claimed.Payload)
		if promptMessageID := payloadInt64(claimed.Payload, "prompt_message_id"); promptMessageID != 0 {
			_ = s.Telegram.DeleteMessage(ctx, message.Chat.ID, promptMessageID)
		}
		text := ui.FormatGoalsSaved(postgres.MessageLink(message.Chat.ID, sourceMessageID, sourceThreadID))
		_, err = s.Telegram.SendMessage(ctx, telegram.SilentMessage(telegram.SendMessageRequest{
			ChatID:          message.Chat.ID,
			MessageThreadID: message.MessageThreadID,
			Text:            text,
			ReplyMarkup:     ui.DismissKeyboard(),
		}))
		return err
	})
}

func (s *Service) consumeGoalWeeklyReview(ctx context.Context, workspace postgres.Workspace, message telegram.Message) error {
	return s.Store.InTx(ctx, func(q *postgres.Queries) error {
		pending, ok, err := q.ClaimPendingInput(ctx, workspace.ID, message.From.ID, message.MessageThreadID, domain.PendingGoalWeeklyReview)
		if err != nil || !ok {
			return err
		}
		reviewID := payloadInt64(pending.Payload, "review_id")
		input := telegram.NewMessageInput(message)
		review, saved, err := q.SubmitGoalWeeklyReview(ctx, reviewID, message.From.ID, input.TextHTML)
		if err != nil {
			return err
		}
		if !saved {
			return errGoalInputNotAccepted
		}
		text := ui.FormatGoalWeeklyReviewSaved(review)
		if !s.editMessageSafe(ctx, message.Chat.ID, payloadInt64(pending.Payload, "prompt_message_id"), text, nil) {
			_, _ = s.Telegram.SendMessage(ctx, telegram.SilentMessage(telegram.SendMessageRequest{
				ChatID:          message.Chat.ID,
				MessageThreadID: message.MessageThreadID,
				Text:            text,
			}))
		}
		return nil
	})
}

func (s *Service) handleGoalFinalStatus(ctx context.Context, callback telegram.CallbackQuery, goalSetID int64, status domain.GoalFinalStatus) (CallbackAnswer, error) {
	workspace, err := s.ensureWorkspaceLoaded(ctx, callback.Message.Chat.ID)
	if err != nil || workspace.ID == 0 {
		return CallbackAnswer{Text: messages.Text("callback.workspace_missing")}, err
	}
	var answer CallbackAnswer
	err = s.Store.InTx(ctx, func(q *postgres.Queries) error {
		goalSet, found, err := q.GetSeasonalGoalSet(ctx, goalSetID)
		if err != nil {
			return err
		}
		if !found {
			answer.Text = messages.Text("goals.not_found")
			return nil
		}
		if goalSet.OwnerUserID != callback.From.ID {
			answer.Text = messages.Text("goals.final.author_only")
			return nil
		}
		if pending, found, err := q.GetPendingInput(ctx, workspace.ID, callback.From.ID, callback.Message.MessageThreadID); err != nil {
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
		_, err = q.UpsertPendingInput(ctx, workspace.ID, callback.From.ID, callback.Message.MessageThreadID, domain.PendingGoalFinalReflection, map[string]any{
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
		pending, ok, err := q.ClaimPendingInput(ctx, workspace.ID, message.From.ID, message.MessageThreadID, domain.PendingGoalFinalReflection)
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
		if err != nil {
			return err
		}
		if !saved {
			return errGoalInputNotAccepted
		}
		text := ui.FormatGoalFinalReviewSaved(goalSet, review)
		if !s.editMessageSafe(ctx, message.Chat.ID, payloadInt64(pending.Payload, "prompt_message_id"), text, nil) {
			_, _ = s.Telegram.SendMessage(ctx, telegram.SilentMessage(telegram.SendMessageRequest{
				ChatID:          message.Chat.ID,
				MessageThreadID: message.MessageThreadID,
				Text:            text,
			}))
		}
		return nil
	})
}

func (s *Service) goalNudge(ctx context.Context, q *postgres.Queries, workspace postgres.Workspace, participant postgres.Participant, seed string, status string) (string, error) {
	return appgoals.MaybeNudge(ctx, q, workspace, participant, seed, status, time.Now().UTC())
}
