package postgres

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/igor/trackmate/internal/domain"
	"github.com/jackc/pgx/v5"
)

const WorkerLockKey int64 = 3_842_001

func (s *Store) TryAcquireWorkerLock(ctx context.Context) (bool, error) {
	var acquired bool
	if err := s.pool.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, WorkerLockKey).Scan(&acquired); err != nil {
		return false, err
	}
	return acquired, nil
}

func (s *Store) ReleaseWorkerLock(ctx context.Context) {
	_, _ = s.pool.Exec(ctx, `SELECT pg_advisory_unlock($1)`, WorkerLockKey)
}

func (q *Queries) ClaimPendingAlert(ctx context.Context) (DailyTaskAlert, bool, error) {
	row := q.db.QueryRow(ctx, `
UPDATE daily_task_alerts
SET dispatch_status = 'dispatching'
WHERE id = (
    SELECT id
    FROM daily_task_alerts
    WHERE dispatch_status = 'pending'
      AND acknowledged_at IS NULL
    ORDER BY id ASC
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING id, daily_task_id, alert_kind::text, dispatch_status::text, telegram_message_id, acknowledged_at, created_at
`)
	alert, err := scanAlert(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return DailyTaskAlert{}, false, nil
	}
	return alert, err == nil, err
}

func (q *Queries) MarkAlertSent(ctx context.Context, alertID int64, messageID int64) error {
	_, err := q.db.Exec(ctx, `
UPDATE daily_task_alerts
SET dispatch_status = 'sent',
    telegram_message_id = $2
WHERE id = $1
`, alertID, messageID)
	return err
}

func (q *Queries) RequeueAlert(ctx context.Context, alertID int64) error {
	_, err := q.db.Exec(ctx, `
UPDATE daily_task_alerts
SET dispatch_status = 'pending'
WHERE id = $1 AND dispatch_status = 'dispatching'
`, alertID)
	return err
}

func (q *Queries) CreateAutoFailProgressEvent(ctx context.Context, task DailyTask, workspace Workspace, participant Participant, todayThreadID int64) error {
	var username any
	var userID any = task.OwnerUserID
	displayName := strconv.FormatInt(task.OwnerUserID, 10)
	if participant.ID != 0 {
		userID = participant.UserID
		displayName = participant.DisplayName
		if participant.Username != nil {
			username = *participant.Username
		}
	}
	payload := map[string]any{
		"task_html":    task.Text,
		"user_id":      userID,
		"display_name": displayName,
		"username":     username,
		"task_link":    MessageLink(workspace.ChatID, optionalInt64(task.TodayCardMessageID), todayThreadID),
	}
	_, err := q.CreateProgressEvent(ctx, task.WorkspaceGroupID, domain.ProgressDailyTaskAutoFail, payload, &task.ParticipantID, &task.ID)
	return err
}

func (q *Queries) CurrentNow(ctx context.Context, fallback time.Time) (time.Time, error) {
	var override *time.Time
	if err := q.db.QueryRow(ctx, `SELECT override_now FROM app_clock WHERE singleton = true`).Scan(&override); err != nil {
		return time.Time{}, err
	}
	if override != nil {
		return override.UTC(), nil
	}
	return fallback.UTC(), nil
}

func (q *Queries) SetClockOverride(ctx context.Context, value *time.Time) error {
	_, err := q.db.Exec(ctx, `
INSERT INTO app_clock (singleton, override_now, updated_at)
VALUES (true, $1, now())
ON CONFLICT (singleton) DO UPDATE SET override_now = EXCLUDED.override_now, updated_at = now()
`, value)
	return err
}
