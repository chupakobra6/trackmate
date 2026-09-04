package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	appdelivery "github.com/igor/trackmate/internal/app/delivery"
	appgoals "github.com/igor/trackmate/internal/app/goals"
	apppending "github.com/igor/trackmate/internal/app/pending"
	appprogress "github.com/igor/trackmate/internal/app/progress"
	approutine "github.com/igor/trackmate/internal/app/routine"
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

var ErrWorkerLockBusy = errors.New("worker lock is busy")

func (r *Runner) Tick(ctx context.Context, now time.Time) (err error) {
	lease, acquired, err := r.Store.TryAcquireWorkerLease(ctx)
	if err != nil || !acquired {
		if err != nil {
			return err
		}
		return ErrWorkerLockBusy
	}
	defer func() {
		err = errors.Join(err, lease.Release())
	}()

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
	if err := approutine.DispatchDueCheckins(ctx, r.Store, r.TG, r.Logger, current); err != nil {
		return err
	}
	if err := approutine.RunCheckinTransitions(ctx, r.Store, r.TG, r.Logger, current); err != nil {
		return err
	}
	if err := approutine.CleanupExpiredNotices(ctx, r.Store, r.TG, current); err != nil {
		return err
	}
	if err := apppending.CleanupStaleInputs(ctx, r.Store, r.TG, current); err != nil {
		return err
	}
	if err := appgoals.DispatchWeeklyReviews(ctx, r.Store, r.TG, current); err != nil {
		return err
	}
	if err := appgoals.DispatchFinalReviews(ctx, r.Store, r.TG, current); err != nil {
		return err
	}
	if err := r.DispatchAlerts(ctx); err != nil {
		return err
	}
	return appprogress.PublishPending(ctx, r.Store, r.TG)
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
		participant, participantFound, err := r.Store.Queries().GetParticipantByID(ctx, task.ParticipantID)
		if err != nil {
			_ = r.Store.Queries().RequeueAlert(ctx, alert.ID)
			return err
		}
		displayName := ""
		username := ""
		userID := task.OwnerUserID
		if participantFound {
			displayName = participant.DisplayName
			username = participantUsername(participant)
			userID = participant.UserID
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
		if alert.AlertKind == domain.AlertOverdueTaskFailed && task.TodayCardMessageID != nil {
			if err := r.TG.EditMessageText(ctx, telegram.EditMessageTextRequest{
				ChatID:      workspace.ChatID,
				MessageID:   *task.TodayCardMessageID,
				Text:        ui.FormatDailyTaskCard(task, displayName, username, ""),
				ReplyMarkup: ui.EmptyKeyboard(),
			}); err != nil && !telegram.IsNotModifiedError(err) && r.Logger != nil {
				r.Logger.WarnContext(ctx, "daily_card_auto_close_edit_failed", "task_id", task.ID, "entry_kind", task.Kind, "error", err)
			}
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
		message, err := r.TG.SendMessage(ctx, telegram.PingMessage(telegram.SendMessageRequest{
			ChatID:           workspace.ChatID,
			MessageThreadID:  todayTopic.ThreadID,
			Text:             ui.DailyEntryAlertText(alert.AlertKind, task.Kind, displayName, username, userID, fmt.Sprintf("daily-alert:%d:%s", alert.ID, alert.AlertKind)),
			ReplyToMessageID: optionalInt64(task.TodayCardMessageID),
			ReplyMarkup:      ui.AlertKeyboard(task.ID, alert.ID),
		}))
		if err != nil {
			_ = r.Store.Queries().RequeueAlert(ctx, alert.ID)
			if r.Logger != nil {
				r.Logger.WarnContext(ctx, "alert_dispatch_failed", "alert_id", alert.ID, "error", err)
			}
			return err
		}
		if err := r.Store.Queries().MarkAlertSent(ctx, alert.ID, message.MessageID); err != nil {
			compensateErr := appdelivery.CompensateSentMessage(r.TG, workspace.ChatID, message.MessageID, func(cleanupCtx context.Context) error {
				return r.Store.Queries().RequeueAlert(cleanupCtx, alert.ID)
			})
			return errors.Join(err, compensateErr)
		}
	}
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
