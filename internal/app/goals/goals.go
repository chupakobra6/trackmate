package goals

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/igor/trackmate/internal/app/delivery"
	"github.com/igor/trackmate/internal/app/messagecleanup"
	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/messages"
	"github.com/igor/trackmate/internal/storage/postgres"
	"github.com/igor/trackmate/internal/telegram"
	"github.com/igor/trackmate/internal/ui"
)

var errWeeklyReviewNoLongerOpen = errors.New("weekly goal review is no longer open")

func MaybeNudge(ctx context.Context, q *postgres.Queries, workspace postgres.Workspace, participant postgres.Participant, seed string, status string, nowFallback time.Time) (string, error) {
	if participant.ID == 0 {
		return "", nil
	}
	now, err := q.CurrentNow(ctx, nowFallback.UTC())
	if err != nil {
		return "", err
	}
	period, err := domain.CurrentGoalPeriod(workspace.Timezone, now)
	if err != nil {
		return "", err
	}
	hasGoals, err := q.HasSeasonalGoalSetForParticipant(ctx, workspace.ID, participant.ID, period.Key)
	if err != nil || !hasGoals {
		return "", err
	}
	cooldown, found, err := q.GetGoalNudgeCooldown(ctx, workspace.ID, participant.ID)
	if err != nil {
		return "", err
	}
	if found && !domain.GoalNudgeAllowed(&cooldown.LastShownAt, now) {
		return "", nil
	}
	if !domain.ShouldShowGoalNudge(fmt.Sprintf("%s:%d:%s", seed, participant.ID, period.Key)) {
		return "", nil
	}
	if err := q.MarkGoalNudgeShown(ctx, workspace.ID, participant.ID, now); err != nil {
		return "", err
	}
	switch status {
	case string(domain.DailyTaskFailed):
		return messages.Text("goal.nudge.failed"), nil
	case string(domain.DailyTaskDone), string(domain.DailyTaskPartial):
		return messages.Text("goal.nudge.done"), nil
	default:
		return messages.Text("goal.nudge.task"), nil
	}
}

func DispatchWeeklyReviews(ctx context.Context, store *postgres.Store, tg telegram.API, nowUTC time.Time) error {
	goalSets, err := store.Queries().ListSeasonalGoalSetContexts(ctx)
	if err != nil {
		return err
	}
	for _, item := range goalSets {
		goalsTopic, found, err := store.Queries().GetTopicBinding(ctx, item.Workspace.ID, domain.TopicGoals)
		if err != nil {
			return err
		}
		if !found {
			continue
		}
		openReview, found, err := store.Queries().GetOpenGoalWeeklyReview(ctx, item.GoalSet.ID)
		if err != nil {
			return err
		}
		if found {
			if err := advanceWeeklyReview(ctx, store, tg, item, goalsTopic, openReview, nowUTC); err != nil {
				return err
			}
			continue
		}
		if !nowBeforeLocalDate(nowUTC, item.Workspace.Timezone, item.GoalSet.PeriodEndsOn) {
			continue
		}
		weekStart, due, err := domain.GoalWeeklyReviewDue(item.GoalSet.PeriodStartsOn, item.Workspace.Timezone, nowUTC)
		if err != nil {
			return err
		}
		if !due {
			continue
		}
		if _, found, err := store.Queries().GetPendingInput(ctx, item.Workspace.ID, item.Participant.UserID, goalsTopic.ThreadID); err != nil {
			return err
		} else if found {
			continue
		}
		review, err := store.Queries().GetOrCreateGoalWeeklyReview(ctx, item.GoalSet.ID, weekStart, nowUTC)
		if err != nil {
			return err
		}
		if review.ResponseText != nil || review.SkippedAt != nil || review.PromptMessageID != nil {
			continue
		}
		if err := sendWeeklyReviewPrompt(ctx, store, tg, item, goalsTopic, review, nowUTC, false); err != nil {
			return err
		}
	}
	return nil
}

func advanceWeeklyReview(ctx context.Context, store *postgres.Store, tg telegram.API, item postgres.SeasonalGoalSetContext, goalsTopic postgres.TopicBinding, review postgres.GoalWeeklyReview, nowUTC time.Time) error {
	action := domain.GoalWeeklyReviewLifecycleAction(review.RequestedAt, review.ReminderSentAt, review.RespondedAt, review.SkippedAt, nowUTC)
	if review.PromptMessageID == nil && review.ReminderSentAt == nil {
		if _, found, err := store.Queries().GetPendingInput(ctx, item.Workspace.ID, item.Participant.UserID, goalsTopic.ThreadID); err != nil {
			return err
		} else if found {
			return nil
		}
		return sendWeeklyReviewPrompt(ctx, store, tg, item, goalsTopic, review, nowUTC, false)
	}
	switch action {
	case domain.GoalWeeklyReviewRemind:
		pending, found, err := store.Queries().GetPendingInput(ctx, item.Workspace.ID, item.Participant.UserID, goalsTopic.ThreadID)
		if err != nil {
			return err
		}
		if found && !pendingBelongsToWeeklyReview(pending, review.ID) {
			return nil
		}
		previousPromptID := review.PromptMessageID
		if err := sendWeeklyReviewPrompt(ctx, store, tg, item, goalsTopic, review, nowUTC, true); err != nil {
			return err
		}
		if previousPromptID != nil {
			closeWeeklyReviewPrompt(ctx, tg, item.Workspace.ChatID, *previousPromptID, review.RequestedAt, nowUTC)
		}
		return nil
	case domain.GoalWeeklyReviewSkip:
		if err := store.InTx(ctx, func(q *postgres.Queries) error {
			skipped, err := q.MarkGoalWeeklyReviewSkipped(ctx, review.ID, nowUTC)
			if err != nil || !skipped {
				return err
			}
			return q.ClearGoalWeeklyReviewPendingInput(ctx, item.Workspace.ID, item.Participant.UserID, goalsTopic.ThreadID, review.ID)
		}); err != nil {
			return err
		}
		if review.PromptMessageID != nil {
			promptSentAt := review.RequestedAt
			if review.ReminderSentAt != nil {
				promptSentAt = *review.ReminderSentAt
			}
			closeWeeklyReviewPrompt(ctx, tg, item.Workspace.ChatID, *review.PromptMessageID, promptSentAt, nowUTC)
		}
		return nil
	default:
		return nil
	}
}

func closeWeeklyReviewPrompt(ctx context.Context, tg telegram.API, chatID int64, messageID int64, sentAt time.Time, nowUTC time.Time) {
	messagecleanup.CloseBestEffort(ctx, tg, chatID, messageID, sentAt, nowUTC, telegram.EditMessageTextRequest{
		Text:        messages.Text("goals.weekly.closed"),
		ReplyMarkup: ui.EmptyKeyboard(),
	})
}

func sendWeeklyReviewPrompt(ctx context.Context, store *postgres.Store, tg telegram.API, item postgres.SeasonalGoalSetContext, goalsTopic postgres.TopicBinding, review postgres.GoalWeeklyReview, nowUTC time.Time, reminder bool) error {
	daysLeft, reviewsLeft, err := domain.GoalReviewCountdown(item.GoalSet.PeriodStartsOn, item.GoalSet.PeriodEndsOn, item.Workspace.Timezone, nowUTC)
	if err != nil {
		return err
	}
	message, err := tg.SendMessage(ctx, telegram.SendMessageRequest{
		ChatID:              item.Workspace.ChatID,
		MessageThreadID:     goalsTopic.ThreadID,
		Text:                ui.FormatGoalWeeklyReviewPrompt(item.GoalSet, item.Participant.DisplayName, participantUsername(item.Participant), goalSourceLink(item.Workspace.ChatID, item.GoalSet), daysLeft, reviewsLeft),
		DisableNotification: true,
	})
	if err != nil {
		return err
	}
	persistErr := store.InTx(ctx, func(q *postgres.Queries) error {
		if reminder {
			updated, err := q.SetGoalWeeklyReviewReminderPrompt(ctx, review.ID, message.MessageID, goalsTopic.ThreadID, nowUTC)
			if err != nil {
				return err
			}
			if !updated {
				return errWeeklyReviewNoLongerOpen
			}
		} else {
			updated, err := q.SetGoalWeeklyReviewPrompt(ctx, review.ID, message.MessageID, goalsTopic.ThreadID, nowUTC)
			if err != nil {
				return err
			}
			if !updated {
				return errWeeklyReviewNoLongerOpen
			}
		}
		_, err := q.UpsertPendingInput(ctx, item.Workspace.ID, item.Participant.UserID, goalsTopic.ThreadID, domain.PendingGoalWeeklyReview, map[string]any{
			"review_id":         review.ID,
			"goal_set_id":       item.GoalSet.ID,
			"prompt_message_id": message.MessageID,
			"thread_id":         goalsTopic.ThreadID,
		})
		return err
	})
	if persistErr == nil {
		return nil
	}
	cleanupErr := delivery.CompensateSentMessage(tg, item.Workspace.ChatID, message.MessageID, nil)
	if errors.Is(persistErr, errWeeklyReviewNoLongerOpen) {
		return cleanupErr
	}
	return errors.Join(persistErr, cleanupErr)
}

func pendingBelongsToWeeklyReview(pending postgres.PendingInput, reviewID int64) bool {
	if pending.Kind != domain.PendingGoalWeeklyReview {
		return false
	}
	switch value := pending.Payload["review_id"].(type) {
	case float64:
		return int64(value) == reviewID
	case int64:
		return value == reviewID
	case int:
		return int64(value) == reviewID
	default:
		return false
	}
}

func DispatchFinalReviews(ctx context.Context, store *postgres.Store, tg telegram.API, nowUTC time.Time) error {
	goalSets, err := store.Queries().ListSeasonalGoalSetContexts(ctx)
	if err != nil {
		return err
	}
	for _, item := range goalSets {
		due, err := domain.GoalFinalReviewDue(domain.GoalPeriod{EndsOn: item.GoalSet.PeriodEndsOn}, item.Workspace.Timezone, nowUTC)
		if err != nil {
			return err
		}
		if !due {
			continue
		}
		goalsTopic, found, err := store.Queries().GetTopicBinding(ctx, item.Workspace.ID, domain.TopicGoals)
		if err != nil {
			return err
		}
		if !found {
			continue
		}
		if _, found, err := store.Queries().GetOpenGoalWeeklyReview(ctx, item.GoalSet.ID); err != nil {
			return err
		} else if found {
			continue
		}
		if _, found, err := store.Queries().GetPendingInput(ctx, item.Workspace.ID, item.Participant.UserID, goalsTopic.ThreadID); err != nil {
			return err
		} else if found {
			continue
		}
		review, err := store.Queries().GetOrCreateGoalFinalReview(ctx, item.GoalSet.ID)
		if err != nil {
			return err
		}
		if review.CompletedAt != nil || review.PromptMessageID != nil {
			continue
		}
		request := telegram.SendMessageRequest{
			ChatID:              item.Workspace.ChatID,
			MessageThreadID:     goalsTopic.ThreadID,
			Text:                ui.FormatGoalFinalReviewPrompt(item.GoalSet, item.Participant.DisplayName, participantUsername(item.Participant), goalSourceLink(item.Workspace.ChatID, item.GoalSet)),
			ReplyMarkup:         ui.GoalFinalStatusKeyboard(item.GoalSet.ID),
			ReplyToMessageID:    goalSourceReplyID(item.GoalSet, goalsTopic.ThreadID),
			DisableNotification: true,
		}
		message, err := tg.SendMessage(ctx, request)
		if err != nil && request.ReplyToMessageID != 0 && telegram.IsMissingReplyTarget(err) {
			request.ReplyToMessageID = 0
			message, err = tg.SendMessage(ctx, request)
		}
		if err != nil {
			return err
		}
		if err := store.Queries().SetGoalFinalReviewPrompt(ctx, review.ID, message.MessageID, goalsTopic.ThreadID); err != nil {
			return errors.Join(err, delivery.CompensateSentMessage(tg, item.Workspace.ChatID, message.MessageID, nil))
		}
	}
	return nil
}

func nowBeforeLocalDate(nowUTC time.Time, timezoneName string, date time.Time) bool {
	location, err := time.LoadLocation(timezoneName)
	if err != nil {
		return false
	}
	localNow := nowUTC.In(location)
	year, month, day := localNow.Date()
	localDate := time.Date(year, month, day, 0, 0, 0, 0, location)
	dateYear, dateMonth, dateDay := date.Date()
	targetDate := time.Date(dateYear, dateMonth, dateDay, 0, 0, 0, 0, location)
	return localDate.Before(targetDate)
}

func goalSourceLink(chatID int64, goalSet postgres.SeasonalGoalSet) string {
	return postgres.MessageLink(chatID, optionalInt64(goalSet.SourceMessageID), optionalInt64(goalSet.SourceMessageThreadID))
}

func goalSourceReplyID(goalSet postgres.SeasonalGoalSet, goalsThreadID int64) int64 {
	if goalSet.SourceMessageID == nil {
		return 0
	}
	if goalSet.SourceMessageThreadID != nil && *goalSet.SourceMessageThreadID != 0 && *goalSet.SourceMessageThreadID != goalsThreadID {
		return 0
	}
	return *goalSet.SourceMessageID
}

func optionalInt64(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func participantUsername(participant postgres.Participant) string {
	if participant.Username == nil {
		return ""
	}
	return *participant.Username
}
