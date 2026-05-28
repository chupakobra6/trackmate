package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/igor/trackmate/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (q *Queries) GetOrCreateWorkspace(ctx context.Context, chatID int64, title string, timezone string) (Workspace, error) {
	row := q.db.QueryRow(ctx, `
INSERT INTO workspace_groups (chat_id, title, timezone, setup_status, created_at, updated_at)
VALUES ($1, NULLIF($2, ''), $3, 'pending', now(), now())
ON CONFLICT (chat_id) DO UPDATE SET
    title = COALESCE(NULLIF(EXCLUDED.title, ''), workspace_groups.title),
    timezone = CASE
        WHEN workspace_groups.timezone IN ('UTC', 'Etc/UTC') THEN EXCLUDED.timezone
        ELSE workspace_groups.timezone
    END,
    updated_at = now()
RETURNING id, chat_id, title, timezone, setup_status::text, setup_message_id, created_at, updated_at
`, chatID, title, timezone)
	return scanWorkspace(row)
}

func (q *Queries) GetWorkspaceByChatID(ctx context.Context, chatID int64) (Workspace, bool, error) {
	row := q.db.QueryRow(ctx, `
SELECT id, chat_id, title, timezone, setup_status::text, setup_message_id, created_at, updated_at
FROM workspace_groups
WHERE chat_id = $1
`, chatID)
	workspace, err := scanWorkspace(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Workspace{}, false, nil
	}
	return workspace, err == nil, err
}

func (q *Queries) GetWorkspaceByID(ctx context.Context, id int64) (Workspace, bool, error) {
	row := q.db.QueryRow(ctx, `
SELECT id, chat_id, title, timezone, setup_status::text, setup_message_id, created_at, updated_at
FROM workspace_groups
WHERE id = $1
`, id)
	workspace, err := scanWorkspace(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Workspace{}, false, nil
	}
	return workspace, err == nil, err
}

func (q *Queries) MarkWorkspaceReady(ctx context.Context, workspaceID int64) error {
	_, err := q.db.Exec(ctx, `
UPDATE workspace_groups
SET setup_status = 'ready', updated_at = now()
WHERE id = $1
`, workspaceID)
	return err
}

func (q *Queries) SetSetupMessageID(ctx context.Context, workspaceID int64, messageID int64) error {
	_, err := q.db.Exec(ctx, `
UPDATE workspace_groups
SET setup_message_id = $2, updated_at = now()
WHERE id = $1
`, workspaceID, messageID)
	return err
}

func (q *Queries) UpsertTopicBinding(ctx context.Context, workspaceID int64, topicKey domain.TopicKey, threadID int64, title string) (TopicBinding, error) {
	row := q.db.QueryRow(ctx, `
INSERT INTO topic_bindings (workspace_group_id, topic_key, thread_id, topic_title, created_at)
VALUES ($1, $2::topickey, $3, $4, now())
ON CONFLICT (workspace_group_id, topic_key) DO UPDATE SET
    thread_id = EXCLUDED.thread_id,
    topic_title = EXCLUDED.topic_title
RETURNING id, workspace_group_id, topic_key::text, thread_id, topic_title, intro_message_id, control_message_id, created_at
`, workspaceID, string(topicKey), threadID, title)
	return scanTopicBinding(row)
}

func (q *Queries) ListTopicBindings(ctx context.Context, workspaceID int64) (map[domain.TopicKey]TopicBinding, error) {
	rows, err := q.db.Query(ctx, `
SELECT id, workspace_group_id, topic_key::text, thread_id, topic_title, intro_message_id, control_message_id, created_at
FROM topic_bindings
WHERE workspace_group_id = $1
ORDER BY id ASC
`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[domain.TopicKey]TopicBinding{}
	for rows.Next() {
		item, err := scanTopicBindingRows(rows)
		if err != nil {
			return nil, err
		}
		result[item.TopicKey] = item
	}
	return result, rows.Err()
}

func (q *Queries) GetTopicBinding(ctx context.Context, workspaceID int64, topicKey domain.TopicKey) (TopicBinding, bool, error) {
	row := q.db.QueryRow(ctx, `
SELECT id, workspace_group_id, topic_key::text, thread_id, topic_title, intro_message_id, control_message_id, created_at
FROM topic_bindings
WHERE workspace_group_id = $1 AND topic_key = $2::topickey
`, workspaceID, string(topicKey))
	item, err := scanTopicBinding(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return TopicBinding{}, false, nil
	}
	return item, err == nil, err
}

func (q *Queries) SetTopicMessages(ctx context.Context, workspaceID int64, topicKey domain.TopicKey, introMessageID *int64, controlMessageID *int64, resetIntro bool, resetControl bool) error {
	_, err := q.db.Exec(ctx, `
UPDATE topic_bindings
SET intro_message_id = CASE WHEN $4 THEN NULL WHEN $2::int IS NULL THEN intro_message_id ELSE $2::int END,
    control_message_id = CASE WHEN $5 THEN NULL WHEN $3::int IS NULL THEN control_message_id ELSE $3::int END
WHERE workspace_group_id = $1 AND topic_key = $6::topickey
`, workspaceID, introMessageID, controlMessageID, resetIntro, resetControl, string(topicKey))
	return err
}

func (q *Queries) RegisterParticipant(ctx context.Context, workspaceID int64, userID int64, username string, displayName string) (Participant, error) {
	row := q.db.QueryRow(ctx, `
INSERT INTO participants (workspace_group_id, user_id, username, display_name, is_active, created_at, updated_at)
VALUES ($1, $2, NULLIF($3, ''), $4, true, now(), now())
ON CONFLICT (workspace_group_id, user_id) DO UPDATE SET
    username = NULLIF(EXCLUDED.username, ''),
    display_name = EXCLUDED.display_name,
    updated_at = now()
RETURNING id, workspace_group_id, user_id, username, display_name, is_active, created_at, updated_at
`, workspaceID, userID, username, displayName)
	return scanParticipant(row)
}

func scanWorkspace(row pgx.Row) (Workspace, error) {
	var item Workspace
	var title pgtype.Text
	var setupMessageID pgtype.Int4
	var status string
	if err := row.Scan(&item.ID, &item.ChatID, &title, &item.Timezone, &status, &setupMessageID, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return Workspace{}, err
	}
	item.Title = textFromPg(title)
	item.SetupMessageID = int64FromPgInt4(setupMessageID)
	item.SetupStatus = domain.GroupSetupStatus(status)
	return item, nil
}

func scanTopicBinding(row pgx.Row) (TopicBinding, error) {
	var item TopicBinding
	var key string
	var intro, control pgtype.Int4
	if err := row.Scan(&item.ID, &item.WorkspaceGroupID, &key, &item.ThreadID, &item.TopicTitle, &intro, &control, &item.CreatedAt); err != nil {
		return TopicBinding{}, err
	}
	item.TopicKey = domain.TopicKey(key)
	item.IntroMessageID = int64FromPgInt4(intro)
	item.ControlMessageID = int64FromPgInt4(control)
	return item, nil
}

func scanTopicBindingRows(rows pgx.Rows) (TopicBinding, error) {
	var item TopicBinding
	var key string
	var intro, control pgtype.Int4
	if err := rows.Scan(&item.ID, &item.WorkspaceGroupID, &key, &item.ThreadID, &item.TopicTitle, &intro, &control, &item.CreatedAt); err != nil {
		return TopicBinding{}, err
	}
	item.TopicKey = domain.TopicKey(key)
	item.IntroMessageID = int64FromPgInt4(intro)
	item.ControlMessageID = int64FromPgInt4(control)
	return item, nil
}

func scanParticipant(row pgx.Row) (Participant, error) {
	var item Participant
	var username pgtype.Text
	if err := row.Scan(&item.ID, &item.WorkspaceGroupID, &item.UserID, &username, &item.DisplayName, &item.IsActive, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return Participant{}, err
	}
	item.Username = textFromPg(username)
	return item, nil
}

func BeginningOfDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}
