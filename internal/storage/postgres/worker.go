package postgres

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/igor/trackmate/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const WorkerLockKey int64 = 3_842_001
const deliveryClaimTimeoutSeconds = 5 * 60
const workerLockReleaseTimeout = 5 * time.Second

var ErrDeliveryClaimLost = errors.New("delivery claim lost")

type WorkerLease struct {
	conn       *pgxpool.Conn
	once       sync.Once
	releaseErr error
}

func (s *Store) TryAcquireWorkerLease(ctx context.Context) (*WorkerLease, bool, error) {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return nil, false, err
	}
	var acquired bool
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1::integer, current_schema()::regnamespace::oid::integer)`, WorkerLockKey).Scan(&acquired); err != nil {
		conn.Release()
		return nil, false, err
	}
	if !acquired {
		conn.Release()
		return nil, false, nil
	}
	return &WorkerLease{conn: conn}, true, nil
}

func (l *WorkerLease) Release() error {
	if l == nil {
		return nil
	}
	l.once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), workerLockReleaseTimeout)
		defer cancel()
		var unlocked bool
		if err := l.conn.QueryRow(ctx, `SELECT pg_advisory_unlock($1::integer, current_schema()::regnamespace::oid::integer)`, WorkerLockKey).Scan(&unlocked); err == nil && unlocked {
			l.conn.Release()
			l.conn = nil
			return
		} else if err != nil {
			l.releaseErr = fmt.Errorf("unlock worker lease: %w", err)
		} else {
			l.releaseErr = errors.New("unlock worker lease: lock was not held by its session")
		}
		raw := l.conn.Hijack()
		l.conn = nil
		if err := raw.Close(ctx); err != nil {
			l.releaseErr = errors.Join(l.releaseErr, fmt.Errorf("close worker lease session: %w", err))
		}
	})
	return l.releaseErr
}

func (q *Queries) ClaimPendingAlert(ctx context.Context) (DailyTaskAlert, bool, error) {
	row := q.db.QueryRow(ctx, `
UPDATE daily_task_alerts
SET dispatch_status = 'dispatching',
    dispatch_started_at = now()
WHERE id = (
    SELECT id
    FROM daily_task_alerts
    WHERE (
            dispatch_status = 'pending'
            OR (
                dispatch_status = 'dispatching'
                AND (dispatch_started_at IS NULL OR dispatch_started_at <= now() - make_interval(secs => $1))
            )
          )
      AND acknowledged_at IS NULL
    ORDER BY id ASC
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING id, daily_task_id, alert_kind::text, dispatch_status::text, telegram_message_id, acknowledged_at, created_at
`, deliveryClaimTimeoutSeconds)
	alert, err := scanAlert(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return DailyTaskAlert{}, false, nil
	}
	return alert, err == nil, err
}

func (q *Queries) MarkAlertSent(ctx context.Context, alertID int64, messageID int64) error {
	tag, err := q.db.Exec(ctx, `
UPDATE daily_task_alerts
SET dispatch_status = 'sent',
    telegram_message_id = $2,
    dispatch_started_at = NULL
WHERE id = $1 AND dispatch_status = 'dispatching'
`, alertID, messageID)
	return requireDeliveryClaim(tag, err, "alert", alertID)
}

func (q *Queries) RequeueAlert(ctx context.Context, alertID int64) error {
	tag, err := q.db.Exec(ctx, `
UPDATE daily_task_alerts
SET dispatch_status = 'pending',
    dispatch_started_at = NULL
WHERE id = $1 AND dispatch_status = 'dispatching'
`, alertID)
	return requireDeliveryClaim(tag, err, "alert", alertID)
}

func requireDeliveryClaim(tag pgconn.CommandTag, err error, kind string, id int64) error {
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("%w: %s %d", ErrDeliveryClaimLost, kind, id)
	}
	return nil
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
		"user_id":      userID,
		"display_name": displayName,
		"username":     username,
	}
	eventType := domain.ProgressDailyTaskAutoFail
	if task.Kind.IsSummary() {
		eventType = domain.ProgressDailySummaryAutoFail
		payload["summary_link"] = MessageLink(workspace.ChatID, optionalInt64(task.TodayCardMessageID), todayThreadID)
	} else {
		payload["task_html"] = task.Text
		payload["task_link"] = MessageLink(workspace.ChatID, optionalInt64(task.TodayCardMessageID), todayThreadID)
	}
	_, err := q.CreateProgressEvent(ctx, task.WorkspaceGroupID, eventType, payload, &task.ParticipantID, &task.ID)
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
