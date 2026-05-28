package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/igor/trackmate/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (q *Queries) CreateProgressEvent(ctx context.Context, workspaceID int64, eventType domain.ProgressEventType, payload map[string]any, participantID *int64, dailyTaskID *int64) (ProgressEvent, error) {
	encoded, err := encodePayload(payload)
	if err != nil {
		return ProgressEvent{}, err
	}
	row := q.db.QueryRow(ctx, `
INSERT INTO progress_events (workspace_group_id, participant_id, daily_task_id, event_type, publish_status, payload, created_at)
VALUES ($1, $2, $3, $4::progresseventtype, 'pending', $5, now())
RETURNING id, workspace_group_id, participant_id, daily_task_id, event_type::text,
          publish_status::text, payload, published_message_id, created_at, published_at
`, workspaceID, participantID, dailyTaskID, string(eventType), encoded)
	return scanProgressEvent(row)
}

func (q *Queries) ClaimProgressEvent(ctx context.Context) (ProgressEvent, bool, error) {
	row := q.db.QueryRow(ctx, `
UPDATE progress_events
SET publish_status = 'publishing'
WHERE id = (
    SELECT id
    FROM progress_events
    WHERE publish_status = 'pending'
    ORDER BY id ASC
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING id, workspace_group_id, participant_id, daily_task_id, event_type::text,
          publish_status::text, payload, published_message_id, created_at, published_at
`)
	event, err := scanProgressEvent(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProgressEvent{}, false, nil
	}
	return event, err == nil, err
}

func (q *Queries) RequeueProgressEvent(ctx context.Context, eventID int64) error {
	_, err := q.db.Exec(ctx, `
UPDATE progress_events
SET publish_status = 'pending'
WHERE id = $1 AND publish_status = 'publishing'
`, eventID)
	return err
}

func (q *Queries) MarkProgressEventFailed(ctx context.Context, eventID int64) error {
	_, err := q.db.Exec(ctx, `
UPDATE progress_events
SET publish_status = 'failed'
WHERE id = $1
`, eventID)
	return err
}

func (q *Queries) MarkProgressEventPublished(ctx context.Context, eventID int64, messageID int64, publishedAt time.Time) error {
	_, err := q.db.Exec(ctx, `
UPDATE progress_events
SET publish_status = 'published',
    published_message_id = $2,
    published_at = $3
WHERE id = $1
`, eventID, messageID, publishedAt.UTC())
	return err
}

func (q *Queries) ListPendingProgressEvents(ctx context.Context) ([]ProgressEvent, error) {
	rows, err := q.db.Query(ctx, `
SELECT id, workspace_group_id, participant_id, daily_task_id, event_type::text,
       publish_status::text, payload, published_message_id, created_at, published_at
FROM progress_events
WHERE publish_status = 'pending'
ORDER BY id ASC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []ProgressEvent
	for rows.Next() {
		event, err := scanProgressEventRows(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func scanProgressEvent(row pgx.Row) (ProgressEvent, error) {
	var event ProgressEvent
	var eventType, publishStatus string
	var participantID, dailyTaskID, messageID pgtype.Int4
	var raw []byte
	var publishedAt pgtype.Timestamptz
	if err := row.Scan(
		&event.ID, &event.WorkspaceGroupID, &participantID, &dailyTaskID,
		&eventType, &publishStatus, &raw, &messageID, &event.CreatedAt, &publishedAt,
	); err != nil {
		return ProgressEvent{}, err
	}
	event.ParticipantID = int64FromPgInt4(participantID)
	event.DailyTaskID = int64FromPgInt4(dailyTaskID)
	event.EventType = domain.ProgressEventType(eventType)
	event.PublishStatus = domain.ProgressPublishStatus(publishStatus)
	event.Payload = decodePayload(raw)
	event.PublishedMessageID = int64FromPgInt4(messageID)
	event.PublishedAt = timeFromPg(publishedAt)
	return event, nil
}

func scanProgressEventRows(rows pgx.Rows) (ProgressEvent, error) {
	var event ProgressEvent
	var eventType, publishStatus string
	var participantID, dailyTaskID, messageID pgtype.Int4
	var raw []byte
	var publishedAt pgtype.Timestamptz
	if err := rows.Scan(
		&event.ID, &event.WorkspaceGroupID, &participantID, &dailyTaskID,
		&eventType, &publishStatus, &raw, &messageID, &event.CreatedAt, &publishedAt,
	); err != nil {
		return ProgressEvent{}, err
	}
	event.ParticipantID = int64FromPgInt4(participantID)
	event.DailyTaskID = int64FromPgInt4(dailyTaskID)
	event.EventType = domain.ProgressEventType(eventType)
	event.PublishStatus = domain.ProgressPublishStatus(publishStatus)
	event.Payload = decodePayload(raw)
	event.PublishedMessageID = int64FromPgInt4(messageID)
	event.PublishedAt = timeFromPg(publishedAt)
	return event, nil
}
