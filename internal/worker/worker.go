package worker

import (
	"context"
	"log/slog"
	"time"

	appprogress "github.com/igor/trackmate/internal/app/progress"
	apptoday "github.com/igor/trackmate/internal/app/today"
	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/storage/postgres"
	"github.com/igor/trackmate/internal/telegram"
	"github.com/igor/trackmate/internal/ui"
)

type Runner struct {
	Store  *postgres.Store
	TG     telegram.API
	Logger *slog.Logger
}

func (r *Runner) Tick(ctx context.Context, now time.Time) error {
	acquired, err := r.Store.TryAcquireWorkerLock(ctx)
	if err != nil || !acquired {
		return err
	}
	defer r.Store.ReleaseWorkerLock(ctx)

	current := now.UTC()
	if err := r.Store.InTx(ctx, func(q *postgres.Queries) error {
		var err error
		current, err = q.CurrentNow(ctx, now.UTC())
		if err != nil {
			return err
		}
		return apptoday.RunDailyTaskTransitions(ctx, q, current)
	}); err != nil {
		return err
	}
	if err := r.DispatchRoutineCheckins(ctx, current); err != nil {
		return err
	}
	if err := r.DispatchGoalWeeklyReviews(ctx, current); err != nil {
		return err
	}
	if err := r.DispatchGoalFinalReviews(ctx, current); err != nil {
		return err
	}
	if err := r.DispatchAlerts(ctx); err != nil {
		return err
	}
	return appprogress.PublishPending(ctx, r.Store, r.TG)
}

func (r *Runner) DispatchRoutineCheckins(ctx context.Context, nowUTC time.Time) error {
	plans, err := r.Store.Queries().ListRoutinePlanContexts(ctx)
	if err != nil {
		return err
	}
	for _, item := range plans {
		checkinDate, due, err := domain.RoutineCheckinDue(item.Plan.CreatedAt, item.Workspace.Timezone, nowUTC)
		if err != nil {
			return err
		}
		if !due || len(item.Plan.Items) == 0 {
			continue
		}
		routineTopic, found, err := r.Store.Queries().GetTopicBinding(ctx, item.Workspace.ID, domain.TopicRoutine)
		if err != nil {
			return err
		}
		if !found {
			continue
		}
		checkin, err := r.Store.Queries().GetOrCreateRoutineCheckin(ctx, item.Plan, checkinDate)
		if err != nil {
			return err
		}
		if checkin.CompletedAt != nil || checkin.CardMessageID != nil {
			continue
		}
		username := ""
		if item.Participant.Username != nil {
			username = *item.Participant.Username
		}
		nextIndex := ui.NextRoutineItemIndex(checkin)
		if nextIndex < 0 {
			continue
		}
		message, err := r.TG.SendMessage(ctx, telegram.SendMessageRequest{
			ChatID:              item.Workspace.ChatID,
			MessageThreadID:     routineTopic.ThreadID,
			Text:                ui.FormatRoutineCheckinCard(checkin, item.Participant.DisplayName, username, ""),
			ReplyMarkup:         ui.RoutineItemKeyboard(checkin.ID, nextIndex),
			DisableNotification: true,
		})
		if err != nil {
			if r.Logger != nil {
				r.Logger.WarnContext(ctx, "routine_checkin_dispatch_failed", "checkin_id", checkin.ID, "error", err)
			}
			return err
		}
		if err := r.Store.Queries().SetRoutineCheckinCardMessageID(ctx, checkin.ID, message.MessageID, routineTopic.ThreadID); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) DispatchGoalWeeklyReviews(ctx context.Context, nowUTC time.Time) error {
	goalSets, err := r.Store.Queries().ListSeasonalGoalSetContexts(ctx)
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
		if _, found, err := r.Store.Queries().GetPendingInput(ctx, item.Workspace.ID, item.Participant.UserID); err != nil {
			return err
		} else if found {
			continue
		}
		review, err := r.Store.Queries().GetOrCreateGoalWeeklyReview(ctx, item.GoalSet.ID, weekStart)
		if err != nil {
			return err
		}
		if review.ResponseText != nil || review.PromptMessageID != nil {
			continue
		}
		goalsTopic, found, err := r.Store.Queries().GetTopicBinding(ctx, item.Workspace.ID, domain.TopicGoals)
		if err != nil {
			return err
		}
		if !found {
			continue
		}
		username := ""
		if item.Participant.Username != nil {
			username = *item.Participant.Username
		}
		message, err := r.TG.SendMessage(ctx, telegram.SendMessageRequest{
			ChatID:              item.Workspace.ChatID,
			MessageThreadID:     goalsTopic.ThreadID,
			Text:                ui.FormatGoalWeeklyReviewPrompt(item.GoalSet, item.Participant.DisplayName, username),
			DisableNotification: true,
		})
		if err != nil {
			return err
		}
		if err := r.Store.Queries().SetGoalWeeklyReviewPrompt(ctx, review.ID, message.MessageID, goalsTopic.ThreadID); err != nil {
			return err
		}
		if _, err := r.Store.Queries().UpsertPendingInput(ctx, item.Workspace.ID, item.Participant.UserID, domain.PendingGoalWeeklyReview, map[string]any{
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

func (r *Runner) DispatchGoalFinalReviews(ctx context.Context, nowUTC time.Time) error {
	goalSets, err := r.Store.Queries().ListSeasonalGoalSetContexts(ctx)
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
		if _, found, err := r.Store.Queries().GetPendingInput(ctx, item.Workspace.ID, item.Participant.UserID); err != nil {
			return err
		} else if found {
			continue
		}
		review, err := r.Store.Queries().GetOrCreateGoalFinalReview(ctx, item.GoalSet.ID)
		if err != nil {
			return err
		}
		if review.CompletedAt != nil || review.PromptMessageID != nil {
			continue
		}
		goalsTopic, found, err := r.Store.Queries().GetTopicBinding(ctx, item.Workspace.ID, domain.TopicGoals)
		if err != nil {
			return err
		}
		if !found {
			continue
		}
		username := ""
		if item.Participant.Username != nil {
			username = *item.Participant.Username
		}
		message, err := r.TG.SendMessage(ctx, telegram.SendMessageRequest{
			ChatID:              item.Workspace.ChatID,
			MessageThreadID:     goalsTopic.ThreadID,
			Text:                ui.FormatGoalFinalReviewPrompt(item.GoalSet, item.Participant.DisplayName, username),
			ReplyMarkup:         ui.GoalFinalStatusKeyboard(item.GoalSet.ID),
			DisableNotification: true,
		})
		if err != nil {
			return err
		}
		if err := r.Store.Queries().SetGoalFinalReviewPrompt(ctx, review.ID, message.MessageID, goalsTopic.ThreadID); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) DispatchAlerts(ctx context.Context) error {
	for {
		alert, ok, err := r.Store.Queries().ClaimPendingAlert(ctx)
		if err != nil || !ok {
			return err
		}
		task, found, err := r.Store.Queries().GetTask(ctx, alert.DailyTaskID)
		if err != nil {
			_ = r.Store.Queries().RequeueAlert(ctx, alert.ID)
			return err
		}
		if !found {
			_ = r.Store.Queries().RequeueAlert(ctx, alert.ID)
			continue
		}
		workspace, found, err := r.Store.Queries().GetWorkspaceByID(ctx, task.WorkspaceGroupID)
		if err != nil {
			_ = r.Store.Queries().RequeueAlert(ctx, alert.ID)
			return err
		}
		if !found {
			_ = r.Store.Queries().RequeueAlert(ctx, alert.ID)
			return nil
		}
		todayTopic, found, err := r.Store.Queries().GetTopicBinding(ctx, workspace.ID, domain.TopicToday)
		if err != nil {
			_ = r.Store.Queries().RequeueAlert(ctx, alert.ID)
			return err
		}
		if !found {
			_ = r.Store.Queries().RequeueAlert(ctx, alert.ID)
			return nil
		}
		message, err := r.TG.SendMessage(ctx, telegram.SendMessageRequest{
			ChatID:              workspace.ChatID,
			MessageThreadID:     todayTopic.ThreadID,
			Text:                ui.AlertText(alert.AlertKind),
			ReplyToMessageID:    optionalInt64(task.TodayCardMessageID),
			ReplyMarkup:         ui.AlertKeyboard(task.ID, alert.ID),
			DisableNotification: true,
		})
		if err != nil {
			_ = r.Store.Queries().RequeueAlert(ctx, alert.ID)
			if r.Logger != nil {
				r.Logger.WarnContext(ctx, "alert_dispatch_failed", "alert_id", alert.ID, "error", err)
			}
			return err
		}
		if err := r.Store.Queries().MarkAlertSent(ctx, alert.ID, message.MessageID); err != nil {
			return err
		}
	}
}

func optionalInt64(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func nowBeforeLocalDate(nowUTC time.Time, timezoneName string, date time.Time) bool {
	location, err := time.LoadLocation(timezoneName)
	if err != nil {
		return false
	}
	localNow := nowUTC.In(location)
	year, month, day := localNow.Date()
	localDate := time.Date(year, month, day, 0, 0, 0, 0, location)
	return localDate.Before(date)
}
