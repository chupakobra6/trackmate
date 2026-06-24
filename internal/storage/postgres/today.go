package postgres

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/igor/trackmate/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (q *Queries) GetOpenTask(ctx context.Context, workspaceID int64, participantID int64) (DailyTask, bool, error) {
	row := q.db.QueryRow(ctx, `
SELECT id, workspace_group_id, participant_id, owner_user_id, task_date, text, status::text,
       report_text, report_status::text, today_card_message_id, created_at, reported_at,
       awaiting_report_at, failed_at, task_message_id, task_message_thread_id,
       report_message_id, report_message_thread_id
FROM daily_tasks
WHERE workspace_group_id = $1
  AND participant_id = $2
  AND status IN ('active', 'awaiting_report')
ORDER BY id DESC
LIMIT 1
`, workspaceID, participantID)
	task, err := scanDailyTask(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return DailyTask{}, false, nil
	}
	return task, err == nil, err
}

func (q *Queries) GetTaskForDate(ctx context.Context, workspaceID int64, participantID int64, taskDate time.Time) (DailyTask, bool, error) {
	row := q.db.QueryRow(ctx, `
SELECT id, workspace_group_id, participant_id, owner_user_id, task_date, text, status::text,
       report_text, report_status::text, today_card_message_id, created_at, reported_at,
       awaiting_report_at, failed_at, task_message_id, task_message_thread_id,
       report_message_id, report_message_thread_id
FROM daily_tasks
WHERE workspace_group_id = $1 AND participant_id = $2 AND task_date = $3::date
`, workspaceID, participantID, taskDate)
	task, err := scanDailyTask(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return DailyTask{}, false, nil
	}
	return task, err == nil, err
}

func (q *Queries) CreateDailyTask(ctx context.Context, workspaceID int64, participantID int64, ownerUserID int64, taskDate time.Time, text string, messageID int64, threadID int64) (DailyTask, bool, error) {
	row := q.db.QueryRow(ctx, `
INSERT INTO daily_tasks (
    workspace_group_id, participant_id, owner_user_id, task_date, text, status,
    task_message_id, task_message_thread_id, created_at
)
VALUES ($1, $2, $3, $4::date, $5, 'active', $6, $7, now())
ON CONFLICT (workspace_group_id, participant_id, task_date) DO NOTHING
RETURNING id, workspace_group_id, participant_id, owner_user_id, task_date, text, status::text,
       report_text, report_status::text, today_card_message_id, created_at, reported_at,
       awaiting_report_at, failed_at, task_message_id, task_message_thread_id,
       report_message_id, report_message_thread_id
`, workspaceID, participantID, ownerUserID, taskDate, text, messageID, threadID)
	task, err := scanDailyTask(row)
	if err == nil {
		return task, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return DailyTask{}, false, err
	}
	existing, found, err := q.GetTaskForDate(ctx, workspaceID, participantID, taskDate)
	return existing, !found, err
}

func (q *Queries) SetDailyTaskCardMessageID(ctx context.Context, taskID int64, messageID int64) error {
	_, err := q.db.Exec(ctx, `
UPDATE daily_tasks
SET today_card_message_id = $2
WHERE id = $1
`, taskID, messageID)
	return err
}

func (q *Queries) GetTask(ctx context.Context, taskID int64) (DailyTask, bool, error) {
	row := q.db.QueryRow(ctx, `
SELECT id, workspace_group_id, participant_id, owner_user_id, task_date, text, status::text,
       report_text, report_status::text, today_card_message_id, created_at, reported_at,
       awaiting_report_at, failed_at, task_message_id, task_message_thread_id,
       report_message_id, report_message_thread_id
FROM daily_tasks
WHERE id = $1
`, taskID)
	task, err := scanDailyTask(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return DailyTask{}, false, nil
	}
	return task, err == nil, err
}

func (q *Queries) ListTasksForTransition(ctx context.Context) ([]DailyTask, error) {
	rows, err := q.db.Query(ctx, `
SELECT id, workspace_group_id, participant_id, owner_user_id, task_date, text, status::text,
       report_text, report_status::text, today_card_message_id, created_at, reported_at,
       awaiting_report_at, failed_at, task_message_id, task_message_thread_id,
       report_message_id, report_message_thread_id
FROM daily_tasks
WHERE status IN ('active', 'awaiting_report')
ORDER BY id ASC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tasks := []DailyTask{}
	for rows.Next() {
		task, err := scanDailyTaskRows(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

func (q *Queries) UpdateTaskAwaitingReport(ctx context.Context, taskID int64, now time.Time) error {
	_, err := q.db.Exec(ctx, `
UPDATE daily_tasks
SET status = 'awaiting_report', awaiting_report_at = $2
WHERE id = $1 AND status = 'active'
`, taskID, now.UTC())
	return err
}

func (q *Queries) UpdateTaskFailed(ctx context.Context, taskID int64, now time.Time) error {
	_, err := q.db.Exec(ctx, `
UPDATE daily_tasks
SET status = 'failed', failed_at = $2
WHERE id = $1 AND status IN ('active', 'awaiting_report')
`, taskID, now.UTC())
	return err
}

func (q *Queries) SubmitTaskReport(ctx context.Context, taskID int64, ownerUserID int64, status domain.DailyTaskStatus, reportHTML string, displayName string, messageID int64, threadID int64) (bool, error) {
	task, found, err := q.GetTask(ctx, taskID)
	if err != nil || !found {
		return false, err
	}
	if task.OwnerUserID != ownerUserID || !task.Status.IsOpen() || !status.IsFinalReport() {
		return false, nil
	}
	_, err = q.db.Exec(ctx, `
UPDATE daily_tasks
SET status = $2::dailytaskstatus,
    report_status = $2::dailytaskstatus,
    report_text = $3,
    report_message_id = $5,
    report_message_thread_id = $6,
    reported_at = now()
WHERE id = $1 AND owner_user_id = $4 AND status IN ('active', 'awaiting_report')
`, taskID, string(status), reportHTML, ownerUserID, messageID, threadID)
	if err != nil {
		return false, err
	}
	participant, _, _ := q.GetParticipantByID(ctx, task.ParticipantID)
	workspace, _, _ := q.GetWorkspaceByID(ctx, task.WorkspaceGroupID)
	todayBinding, hasToday, _ := q.GetTopicBinding(ctx, task.WorkspaceGroupID, domain.TopicToday)
	var username any
	var participantUserID any = ownerUserID
	if participant.ID != 0 {
		participantUserID = participant.UserID
		if participant.Username != nil {
			username = *participant.Username
		}
	}
	payload := map[string]any{
		"status":       string(status),
		"report_html":  reportHTML,
		"user_id":      participantUserID,
		"display_name": displayName,
		"username":     username,
		"task_html":    task.Text,
	}
	if workspace.ID != 0 {
		var threadID int64
		if hasToday {
			threadID = todayBinding.ThreadID
		}
		payload["task_link"] = MessageLink(workspace.ChatID, optionalInt64(task.TodayCardMessageID), threadID)
	}
	if _, err := q.CreateProgressEvent(ctx, task.WorkspaceGroupID, domain.ProgressDailyTaskClosed, payload, &task.ParticipantID, &task.ID); err != nil {
		return false, err
	}
	return true, nil
}

func (q *Queries) UpdateTaskTextFromSourceMessage(ctx context.Context, workspaceID int64, ownerUserID int64, messageID int64, threadID int64, text string) (DailyTask, []ProgressEvent, bool, error) {
	row := q.db.QueryRow(ctx, `
UPDATE daily_tasks
SET text = $5
WHERE workspace_group_id = $1
  AND owner_user_id = $2
  AND task_message_id = $3
  AND task_message_thread_id = $4
RETURNING id, workspace_group_id, participant_id, owner_user_id, task_date, text, status::text,
       report_text, report_status::text, today_card_message_id, created_at, reported_at,
       awaiting_report_at, failed_at, task_message_id, task_message_thread_id,
       report_message_id, report_message_thread_id
`, workspaceID, ownerUserID, messageID, threadID, text)
	task, err := scanDailyTask(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return DailyTask{}, nil, false, nil
	}
	if err != nil {
		return DailyTask{}, nil, false, err
	}
	events, err := q.SyncDailyTaskProgressPayloads(ctx, task)
	if err != nil {
		return DailyTask{}, nil, false, err
	}
	return task, events, true, nil
}

func (q *Queries) UpdateTaskReportFromSourceMessage(ctx context.Context, workspaceID int64, ownerUserID int64, messageID int64, threadID int64, reportHTML string) (DailyTask, []ProgressEvent, bool, error) {
	row := q.db.QueryRow(ctx, `
UPDATE daily_tasks
SET report_text = $5
WHERE workspace_group_id = $1
  AND owner_user_id = $2
  AND report_message_id = $3
  AND report_message_thread_id = $4
  AND report_text IS NOT NULL
RETURNING id, workspace_group_id, participant_id, owner_user_id, task_date, text, status::text,
       report_text, report_status::text, today_card_message_id, created_at, reported_at,
       awaiting_report_at, failed_at, task_message_id, task_message_thread_id,
       report_message_id, report_message_thread_id
`, workspaceID, ownerUserID, messageID, threadID, reportHTML)
	task, err := scanDailyTask(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return DailyTask{}, nil, false, nil
	}
	if err != nil {
		return DailyTask{}, nil, false, err
	}
	events, err := q.SyncDailyTaskProgressPayloads(ctx, task)
	if err != nil {
		return DailyTask{}, nil, false, err
	}
	return task, events, true, nil
}

func (q *Queries) SyncDailyTaskProgressPayloads(ctx context.Context, task DailyTask) ([]ProgressEvent, error) {
	taskLink := ""
	workspace, found, err := q.GetWorkspaceByID(ctx, task.WorkspaceGroupID)
	if err != nil {
		return nil, err
	}
	if found {
		todayBinding, hasToday, err := q.GetTopicBinding(ctx, task.WorkspaceGroupID, domain.TopicToday)
		if err != nil {
			return nil, err
		}
		var threadID int64
		if hasToday {
			threadID = todayBinding.ThreadID
		}
		taskLink = MessageLink(workspace.ChatID, optionalInt64(task.TodayCardMessageID), threadID)
	}
	reportHTML := ""
	if task.ReportText != nil {
		reportHTML = *task.ReportText
	}
	rows, err := q.db.Query(ctx, `
UPDATE progress_events
SET payload = CASE
    WHEN event_type = 'daily_task.closed'::progresseventtype THEN
        payload::jsonb || jsonb_build_object('task_html', $2::text, 'report_html', $3::text, 'task_link', $4::text)
    WHEN event_type = 'daily_task.auto_failed'::progresseventtype THEN
        payload::jsonb || jsonb_build_object('task_html', $2::text, 'task_link', $4::text)
    ELSE payload
END
WHERE daily_task_id = $1
  AND event_type IN ('daily_task.closed'::progresseventtype, 'daily_task.auto_failed'::progresseventtype)
RETURNING id, workspace_group_id, participant_id, daily_task_id, event_type::text,
          publish_status::text, payload, published_message_id, created_at, published_at
`, task.ID, task.Text, reportHTML, taskLink)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []ProgressEvent{}
	for rows.Next() {
		event, err := scanProgressEventRows(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (q *Queries) GetParticipantByID(ctx context.Context, participantID int64) (Participant, bool, error) {
	row := q.db.QueryRow(ctx, `
SELECT id, workspace_group_id, user_id, username, display_name, is_active, created_at, updated_at
FROM participants
WHERE id = $1
`, participantID)
	item, err := scanParticipant(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Participant{}, false, nil
	}
	return item, err == nil, err
}

func (q *Queries) GetOrCreateAlert(ctx context.Context, taskID int64, kind domain.AlertKind) (DailyTaskAlert, error) {
	row := q.db.QueryRow(ctx, `
INSERT INTO daily_task_alerts (daily_task_id, alert_kind, dispatch_status, created_at)
VALUES ($1, $2::alertkind, 'pending', now())
ON CONFLICT (daily_task_id, alert_kind) DO UPDATE SET daily_task_id = EXCLUDED.daily_task_id
RETURNING id, daily_task_id, alert_kind::text, dispatch_status::text, telegram_message_id, acknowledged_at, created_at
`, taskID, string(kind))
	return scanAlert(row)
}

func (q *Queries) ListAlertsForTask(ctx context.Context, taskID int64) ([]DailyTaskAlert, error) {
	rows, err := q.db.Query(ctx, `
SELECT id, daily_task_id, alert_kind::text, dispatch_status::text, telegram_message_id, acknowledged_at, created_at
FROM daily_task_alerts
WHERE daily_task_id = $1
ORDER BY id ASC
`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var alerts []DailyTaskAlert
	for rows.Next() {
		alert, err := scanAlertRows(rows)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}
	return alerts, rows.Err()
}

func (q *Queries) AcknowledgeAlert(ctx context.Context, alertID int64, now time.Time) error {
	_, err := q.db.Exec(ctx, `
UPDATE daily_task_alerts
SET acknowledged_at = COALESCE(acknowledged_at, $2), telegram_message_id = NULL
WHERE id = $1
`, alertID, now.UTC())
	return err
}

func (q *Queries) ClearAlertMessage(ctx context.Context, alertID int64) error {
	_, err := q.db.Exec(ctx, `
UPDATE daily_task_alerts SET telegram_message_id = NULL WHERE id = $1
`, alertID)
	return err
}

func (q *Queries) GetAlert(ctx context.Context, alertID int64) (DailyTaskAlert, bool, error) {
	row := q.db.QueryRow(ctx, `
SELECT id, daily_task_id, alert_kind::text, dispatch_status::text, telegram_message_id, acknowledged_at, created_at
FROM daily_task_alerts
WHERE id = $1
`, alertID)
	alert, err := scanAlert(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return DailyTaskAlert{}, false, nil
	}
	return alert, err == nil, err
}

func (q *Queries) GetPendingInput(ctx context.Context, workspaceID int64, userID int64, threadID int64) (PendingInput, bool, error) {
	row := q.db.QueryRow(ctx, `
SELECT id, workspace_group_id, user_id, message_thread_id, kind, payload, created_at
FROM pending_inputs
WHERE workspace_group_id = $1 AND user_id = $2 AND message_thread_id = $3
`, workspaceID, userID, threadID)
	pending, err := scanPending(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return PendingInput{}, false, nil
	}
	return pending, err == nil, err
}

func (q *Queries) UpsertPendingInput(ctx context.Context, workspaceID int64, userID int64, threadID int64, kind domain.PendingInputKind, payload map[string]any) (PendingInput, error) {
	payloadWithThread := copyPayload(payload)
	payloadWithThread["thread_id"] = threadID
	encoded, err := encodePayload(payloadWithThread)
	if err != nil {
		return PendingInput{}, err
	}
	row := q.db.QueryRow(ctx, `
INSERT INTO pending_inputs (workspace_group_id, user_id, message_thread_id, kind, payload, created_at)
VALUES ($1, $2, $3, $4, $5, now())
ON CONFLICT (workspace_group_id, user_id, message_thread_id) DO UPDATE SET
    kind = EXCLUDED.kind,
    payload = EXCLUDED.payload,
    created_at = now()
RETURNING id, workspace_group_id, user_id, message_thread_id, kind, payload, created_at
`, workspaceID, userID, threadID, string(kind), encoded)
	return scanPending(row)
}

func (q *Queries) ClaimPendingInput(ctx context.Context, workspaceID int64, userID int64, threadID int64, kind domain.PendingInputKind) (PendingInput, bool, error) {
	row := q.db.QueryRow(ctx, `
DELETE FROM pending_inputs
WHERE workspace_group_id = $1 AND user_id = $2 AND message_thread_id = $3 AND kind = $4
RETURNING id, workspace_group_id, user_id, message_thread_id, kind, payload, created_at
`, workspaceID, userID, threadID, string(kind))
	pending, err := scanPending(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return PendingInput{}, false, nil
	}
	return pending, err == nil, err
}

func (q *Queries) ClearPendingInput(ctx context.Context, workspaceID int64, userID int64, threadID int64) error {
	_, err := q.db.Exec(ctx, `
DELETE FROM pending_inputs
WHERE workspace_group_id = $1 AND user_id = $2 AND message_thread_id = $3
`, workspaceID, userID, threadID)
	return err
}

func (q *Queries) ClearRoutineCheckinPendingInput(ctx context.Context, workspaceID int64, userID int64, threadID int64, checkinID int64) error {
	_, err := q.db.Exec(ctx, `
DELETE FROM pending_inputs
WHERE workspace_group_id = $1
  AND user_id = $2
  AND message_thread_id = $3
  AND payload->>'checkin_id' = $4
`, workspaceID, userID, threadID, fmt.Sprint(checkinID))
	return err
}

func (q *Queries) ListStalePendingInputContexts(ctx context.Context, cutoff time.Time, limit int) ([]PendingInputContext, error) {
	rows, err := q.db.Query(ctx, `
SELECT pi.id, pi.workspace_group_id, pi.user_id, pi.message_thread_id, pi.kind, pi.payload, pi.created_at,
       wg.id, wg.chat_id, wg.title, wg.timezone, wg.setup_status::text, wg.setup_message_id, wg.created_at, wg.updated_at
FROM pending_inputs pi
JOIN workspace_groups wg ON wg.id = pi.workspace_group_id
WHERE pi.created_at <= $1
ORDER BY pi.created_at ASC
LIMIT $2
`, cutoff.UTC(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []PendingInputContext
	for rows.Next() {
		var item PendingInputContext
		var kind string
		var raw []byte
		var title pgtype.Text
		var setupStatus string
		var setupMessageID pgtype.Int4
		if err := rows.Scan(
			&item.Pending.ID, &item.Pending.WorkspaceGroupID, &item.Pending.UserID, &item.Pending.MessageThreadID, &kind, &raw, &item.Pending.CreatedAt,
			&item.Workspace.ID, &item.Workspace.ChatID, &title, &item.Workspace.Timezone, &setupStatus, &setupMessageID, &item.Workspace.CreatedAt, &item.Workspace.UpdatedAt,
		); err != nil {
			return nil, err
		}
		item.Pending.Kind = domain.PendingInputKind(kind)
		item.Pending.Payload = decodePayload(raw)
		item.Workspace.Title = textFromPg(title)
		item.Workspace.SetupStatus = domain.GroupSetupStatus(setupStatus)
		item.Workspace.SetupMessageID = int64FromPgInt4(setupMessageID)
		result = append(result, item)
	}
	return result, rows.Err()
}

func (q *Queries) ClaimStalePendingInput(ctx context.Context, pendingID int64, cutoff time.Time) (PendingInput, bool, error) {
	row := q.db.QueryRow(ctx, `
DELETE FROM pending_inputs
WHERE id = $1 AND created_at <= $2
RETURNING id, workspace_group_id, user_id, message_thread_id, kind, payload, created_at
`, pendingID, cutoff.UTC())
	pending, err := scanPending(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return PendingInput{}, false, nil
	}
	return pending, err == nil, err
}

func scanDailyTask(row pgx.Row) (DailyTask, error) {
	var task DailyTask
	var status string
	var reportText, reportStatus pgtype.Text
	var card, taskMessageID, taskThreadID, reportMessageID, reportThreadID pgtype.Int4
	var reportedAt, awaitingAt, failedAt pgtype.Timestamptz
	if err := row.Scan(
		&task.ID, &task.WorkspaceGroupID, &task.ParticipantID, &task.OwnerUserID, &task.TaskDate,
		&task.Text, &status, &reportText, &reportStatus, &card, &task.CreatedAt, &reportedAt, &awaitingAt, &failedAt,
		&taskMessageID, &taskThreadID, &reportMessageID, &reportThreadID,
	); err != nil {
		return DailyTask{}, err
	}
	task.Status = domain.DailyTaskStatus(status)
	task.ReportText = textFromPg(reportText)
	task.ReportStatus = statusFromPg(reportStatus)
	task.TodayCardMessageID = int64FromPgInt4(card)
	task.TaskMessageID = int64FromPgInt4(taskMessageID)
	task.TaskMessageThreadID = int64FromPgInt4(taskThreadID)
	task.ReportMessageID = int64FromPgInt4(reportMessageID)
	task.ReportMessageThreadID = int64FromPgInt4(reportThreadID)
	task.ReportedAt = timeFromPg(reportedAt)
	task.AwaitingReportAt = timeFromPg(awaitingAt)
	task.FailedAt = timeFromPg(failedAt)
	return task, nil
}

func scanDailyTaskRows(rows pgx.Rows) (DailyTask, error) {
	var task DailyTask
	var status string
	var reportText, reportStatus pgtype.Text
	var card, taskMessageID, taskThreadID, reportMessageID, reportThreadID pgtype.Int4
	var reportedAt, awaitingAt, failedAt pgtype.Timestamptz
	if err := rows.Scan(
		&task.ID, &task.WorkspaceGroupID, &task.ParticipantID, &task.OwnerUserID, &task.TaskDate,
		&task.Text, &status, &reportText, &reportStatus, &card, &task.CreatedAt, &reportedAt, &awaitingAt, &failedAt,
		&taskMessageID, &taskThreadID, &reportMessageID, &reportThreadID,
	); err != nil {
		return DailyTask{}, err
	}
	task.Status = domain.DailyTaskStatus(status)
	task.ReportText = textFromPg(reportText)
	task.ReportStatus = statusFromPg(reportStatus)
	task.TodayCardMessageID = int64FromPgInt4(card)
	task.TaskMessageID = int64FromPgInt4(taskMessageID)
	task.TaskMessageThreadID = int64FromPgInt4(taskThreadID)
	task.ReportMessageID = int64FromPgInt4(reportMessageID)
	task.ReportMessageThreadID = int64FromPgInt4(reportThreadID)
	task.ReportedAt = timeFromPg(reportedAt)
	task.AwaitingReportAt = timeFromPg(awaitingAt)
	task.FailedAt = timeFromPg(failedAt)
	return task, nil
}

func scanAlert(row pgx.Row) (DailyTaskAlert, error) {
	var alert DailyTaskAlert
	var kind, status string
	var messageID pgtype.Int4
	var acknowledged pgtype.Timestamptz
	if err := row.Scan(&alert.ID, &alert.DailyTaskID, &kind, &status, &messageID, &acknowledged, &alert.CreatedAt); err != nil {
		return DailyTaskAlert{}, err
	}
	alert.AlertKind = domain.AlertKind(kind)
	alert.DispatchStatus = domain.AlertDispatchStatus(status)
	alert.TelegramMessageID = int64FromPgInt4(messageID)
	alert.AcknowledgedAt = timeFromPg(acknowledged)
	return alert, nil
}

func scanAlertRows(rows pgx.Rows) (DailyTaskAlert, error) {
	var alert DailyTaskAlert
	var kind, status string
	var messageID pgtype.Int4
	var acknowledged pgtype.Timestamptz
	if err := rows.Scan(&alert.ID, &alert.DailyTaskID, &kind, &status, &messageID, &acknowledged, &alert.CreatedAt); err != nil {
		return DailyTaskAlert{}, err
	}
	alert.AlertKind = domain.AlertKind(kind)
	alert.DispatchStatus = domain.AlertDispatchStatus(status)
	alert.TelegramMessageID = int64FromPgInt4(messageID)
	alert.AcknowledgedAt = timeFromPg(acknowledged)
	return alert, nil
}

func scanPending(row pgx.Row) (PendingInput, error) {
	var pending PendingInput
	var kind string
	var raw []byte
	if err := row.Scan(&pending.ID, &pending.WorkspaceGroupID, &pending.UserID, &pending.MessageThreadID, &kind, &raw, &pending.CreatedAt); err != nil {
		return PendingInput{}, err
	}
	pending.Kind = domain.PendingInputKind(kind)
	pending.Payload = decodePayload(raw)
	return pending, nil
}

func optionalInt64(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func MessageLink(chatID int64, messageID int64, threadID int64) string {
	if messageID == 0 {
		return ""
	}
	chatText := strconv.FormatInt(chatID, 10)
	if !strings.HasPrefix(chatText, "-100") {
		return ""
	}
	link := "https://t.me/c/" + chatText[4:] + "/" + strconv.FormatInt(messageID, 10)
	if threadID != 0 {
		link += "?thread=" + strconv.FormatInt(threadID, 10)
	}
	return link
}
