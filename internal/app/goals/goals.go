package goals

import (
	"context"
	"fmt"
	"time"

	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/messages"
	"github.com/igor/trackmate/internal/storage/postgres"
	"github.com/igor/trackmate/internal/telegram"
	"github.com/igor/trackmate/internal/ui"
)

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
		if !nowBeforeLocalDate(nowUTC, item.Workspace.Timezone, item.GoalSet.PeriodEndsOn) {
			continue
		}
		weekStart, due, err := domain.GoalWeeklyReviewDue(item.Workspace.Timezone, nowUTC)
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
		if _, found, err := store.Queries().GetPendingInput(ctx, item.Workspace.ID, item.Participant.UserID, goalsTopic.ThreadID); err != nil {
			return err
		} else if found {
			continue
		}
		review, err := store.Queries().GetOrCreateGoalWeeklyReview(ctx, item.GoalSet.ID, weekStart)
		if err != nil {
			return err
		}
		if review.ResponseText != nil || review.PromptMessageID != nil {
			continue
		}
		message, err := tg.SendMessage(ctx, telegram.SendMessageRequest{
			ChatID:              item.Workspace.ChatID,
			MessageThreadID:     goalsTopic.ThreadID,
			Text:                ui.FormatGoalWeeklyReviewPrompt(item.GoalSet, item.Participant.DisplayName, participantUsername(item.Participant)),
			DisableNotification: true,
		})
		if err != nil {
			return err
		}
		if err := store.Queries().SetGoalWeeklyReviewPrompt(ctx, review.ID, message.MessageID, goalsTopic.ThreadID); err != nil {
			return err
		}
		if _, err := store.Queries().UpsertPendingInput(ctx, item.Workspace.ID, item.Participant.UserID, goalsTopic.ThreadID, domain.PendingGoalWeeklyReview, map[string]any{
			"review_id":         review.ID,
			"goal_set_id":       item.GoalSet.ID,
			"prompt_message_id": message.MessageID,
			"thread_id":         goalsTopic.ThreadID,
		}); err != nil {
			return err
		}
	}
	return nil
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
		message, err := tg.SendMessage(ctx, telegram.SendMessageRequest{
			ChatID:              item.Workspace.ChatID,
			MessageThreadID:     goalsTopic.ThreadID,
			Text:                ui.FormatGoalFinalReviewPrompt(item.GoalSet, item.Participant.DisplayName, participantUsername(item.Participant)),
			ReplyMarkup:         ui.GoalFinalStatusKeyboard(item.GoalSet.ID),
			DisableNotification: true,
		})
		if err != nil {
			return err
		}
		if err := store.Queries().SetGoalFinalReviewPrompt(ctx, review.ID, message.MessageID, goalsTopic.ThreadID); err != nil {
			return err
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
	dateYear, dateMonth, dateDay := date.In(location).Date()
	targetDate := time.Date(dateYear, dateMonth, dateDay, 0, 0, 0, 0, location)
	return localDate.Before(targetDate)
}

func participantUsername(participant postgres.Participant) string {
	if participant.Username == nil {
		return ""
	}
	return *participant.Username
}
