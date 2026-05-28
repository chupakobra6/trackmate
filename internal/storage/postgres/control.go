package postgres

import (
	"context"

	"github.com/igor/trackmate/internal/domain"
)

type ResetWorkspaceResult struct {
	WorkspaceID     int64 `json:"workspace_id"`
	ChatID          int64 `json:"chat_id"`
	DeletedTasks    int64 `json:"deleted_tasks"`
	DeletedAlerts   int64 `json:"deleted_alerts"`
	DeletedPending  int64 `json:"deleted_pending_inputs"`
	DeletedProgress int64 `json:"deleted_progress_events"`
	ResetTopics     int64 `json:"reset_topic_messages"`
}

func (q *Queries) ResetWorkspaceForE2E(ctx context.Context, chatID int64) (ResetWorkspaceResult, error) {
	workspace, found, err := q.GetWorkspaceByChatID(ctx, chatID)
	if err != nil || !found {
		return ResetWorkspaceResult{ChatID: chatID}, err
	}
	result := ResetWorkspaceResult{WorkspaceID: workspace.ID, ChatID: chatID}
	tag, err := q.db.Exec(ctx, `DELETE FROM pending_inputs WHERE workspace_group_id = $1`, workspace.ID)
	if err != nil {
		return result, err
	}
	result.DeletedPending = tag.RowsAffected()
	tag, err = q.db.Exec(ctx, `DELETE FROM daily_task_alerts WHERE daily_task_id IN (SELECT id FROM daily_tasks WHERE workspace_group_id = $1)`, workspace.ID)
	if err != nil {
		return result, err
	}
	result.DeletedAlerts = tag.RowsAffected()
	tag, err = q.db.Exec(ctx, `DELETE FROM progress_events WHERE workspace_group_id = $1`, workspace.ID)
	if err != nil {
		return result, err
	}
	result.DeletedProgress = tag.RowsAffected()
	tag, err = q.db.Exec(ctx, `DELETE FROM daily_tasks WHERE workspace_group_id = $1`, workspace.ID)
	if err != nil {
		return result, err
	}
	result.DeletedTasks = tag.RowsAffected()
	_, err = q.db.Exec(ctx, `
UPDATE workspace_groups
SET setup_status = 'pending',
    updated_at = now()
WHERE id = $1
`, workspace.ID)
	return result, err
}

func (q *Queries) ActiveTopicBindings(ctx context.Context, chatID int64) (map[domain.TopicKey]TopicBinding, error) {
	workspace, found, err := q.GetWorkspaceByChatID(ctx, chatID)
	if err != nil || !found {
		return map[domain.TopicKey]TopicBinding{}, err
	}
	return q.ListTopicBindings(ctx, workspace.ID)
}
