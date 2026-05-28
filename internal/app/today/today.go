package today

import (
	"context"
	"time"

	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/storage/postgres"
)

func RunDailyTaskTransitions(ctx context.Context, q *postgres.Queries, nowUTC time.Time) error {
	tasks, err := q.ListTasksForTransition(ctx)
	if err != nil {
		return err
	}
	for _, task := range tasks {
		workspace, found, err := q.GetWorkspaceByID(ctx, task.WorkspaceGroupID)
		if err != nil || !found {
			return err
		}
		transition, err := domain.NextDailyTaskTransition(task.TaskDate, workspace.Timezone, task.Status, nowUTC)
		if err != nil {
			return err
		}
		switch transition.NewStatus {
		case domain.DailyTaskAwaitingReport:
			if err := q.UpdateTaskAwaitingReport(ctx, task.ID, nowUTC); err != nil {
				return err
			}
			if _, err := q.GetOrCreateAlert(ctx, task.ID, domain.AlertDayClosedPendingReport); err != nil {
				return err
			}
		case domain.DailyTaskFailed:
			if err := q.UpdateTaskFailed(ctx, task.ID, nowUTC); err != nil {
				return err
			}
			if _, err := q.GetOrCreateAlert(ctx, task.ID, domain.AlertOverdueTaskFailed); err != nil {
				return err
			}
			participant, _, err := q.GetParticipantByID(ctx, task.ParticipantID)
			if err != nil {
				return err
			}
			todayBinding, _, err := q.GetTopicBinding(ctx, task.WorkspaceGroupID, domain.TopicToday)
			if err != nil {
				return err
			}
			if err := q.CreateAutoFailProgressEvent(ctx, task, workspace, participant, todayBinding.ThreadID); err != nil {
				return err
			}
		}
	}
	return nil
}
