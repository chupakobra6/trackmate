-- +goose Up
-- +goose StatementBegin
DO $$ BEGIN
    CREATE TYPE groupsetupstatus AS ENUM ('pending', 'requirements_failed', 'ready');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE topickey AS ENUM ('today', 'progress');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE dailytaskstatus AS ENUM ('active', 'awaiting_report', 'done', 'partial', 'failed');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE alertkind AS ENUM ('day_closed_pending_report', 'overdue_task_failed');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE progresseventtype AS ENUM ('daily_task.closed', 'daily_task.auto_failed', 'system_alert', 'custom_update');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

ALTER TYPE progresseventtype ADD VALUE IF NOT EXISTS 'custom_update';

DO $$ BEGIN
    CREATE TYPE progresspublishstatus AS ENUM ('pending', 'publishing', 'published', 'failed');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE alertdispatchstatus AS ENUM ('pending', 'dispatching', 'sent');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

CREATE TABLE IF NOT EXISTS workspace_groups (
    id SERIAL PRIMARY KEY,
    chat_id BIGINT NOT NULL UNIQUE,
    title VARCHAR(255),
    timezone VARCHAR(64) NOT NULL DEFAULT 'UTC',
    setup_status groupsetupstatus NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    setup_message_id INTEGER
);
CREATE INDEX IF NOT EXISTS ix_workspace_groups_chat_id ON workspace_groups(chat_id);

CREATE TABLE IF NOT EXISTS topic_bindings (
    id SERIAL PRIMARY KEY,
    workspace_group_id INTEGER NOT NULL REFERENCES workspace_groups(id) ON DELETE CASCADE,
    topic_key topickey NOT NULL,
    thread_id INTEGER NOT NULL,
    topic_title VARCHAR(255) NOT NULL,
    intro_message_id INTEGER,
    control_message_id INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_group_id, topic_key)
);
CREATE INDEX IF NOT EXISTS ix_topic_bindings_workspace_group_id ON topic_bindings(workspace_group_id);
CREATE INDEX IF NOT EXISTS ix_topic_bindings_thread_id ON topic_bindings(thread_id);

CREATE TABLE IF NOT EXISTS participants (
    id SERIAL PRIMARY KEY,
    workspace_group_id INTEGER NOT NULL REFERENCES workspace_groups(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL,
    username VARCHAR(255),
    display_name VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_group_id, user_id)
);
CREATE INDEX IF NOT EXISTS ix_participants_workspace_group_id ON participants(workspace_group_id);
CREATE INDEX IF NOT EXISTS ix_participants_user_id ON participants(user_id);

CREATE TABLE IF NOT EXISTS daily_tasks (
    id SERIAL PRIMARY KEY,
    workspace_group_id INTEGER NOT NULL REFERENCES workspace_groups(id) ON DELETE CASCADE,
    participant_id INTEGER NOT NULL REFERENCES participants(id) ON DELETE CASCADE,
    owner_user_id BIGINT NOT NULL,
    task_date DATE NOT NULL,
    text TEXT NOT NULL,
    status dailytaskstatus NOT NULL DEFAULT 'active',
    report_text TEXT,
    report_status dailytaskstatus,
    today_card_message_id INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    reported_at TIMESTAMPTZ,
    awaiting_report_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    UNIQUE (workspace_group_id, participant_id, task_date)
);
CREATE INDEX IF NOT EXISTS ix_daily_tasks_workspace_group_id ON daily_tasks(workspace_group_id);
CREATE INDEX IF NOT EXISTS ix_daily_tasks_participant_id ON daily_tasks(participant_id);
CREATE INDEX IF NOT EXISTS ix_daily_tasks_owner_user_id ON daily_tasks(owner_user_id);
CREATE INDEX IF NOT EXISTS ix_daily_tasks_status ON daily_tasks(status);

CREATE TABLE IF NOT EXISTS progress_events (
    id SERIAL PRIMARY KEY,
    workspace_group_id INTEGER NOT NULL REFERENCES workspace_groups(id) ON DELETE CASCADE,
    participant_id INTEGER REFERENCES participants(id) ON DELETE SET NULL,
    daily_task_id INTEGER REFERENCES daily_tasks(id) ON DELETE SET NULL,
    event_type progresseventtype NOT NULL,
    publish_status progresspublishstatus NOT NULL DEFAULT 'pending',
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    published_message_id INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS ix_progress_events_workspace_group_id ON progress_events(workspace_group_id);
CREATE INDEX IF NOT EXISTS ix_progress_events_participant_id ON progress_events(participant_id);
CREATE INDEX IF NOT EXISTS ix_progress_events_daily_task_id ON progress_events(daily_task_id);
CREATE INDEX IF NOT EXISTS ix_progress_events_event_type ON progress_events(event_type);

CREATE TABLE IF NOT EXISTS daily_task_alerts (
    id SERIAL PRIMARY KEY,
    daily_task_id INTEGER NOT NULL REFERENCES daily_tasks(id) ON DELETE CASCADE,
    alert_kind alertkind NOT NULL,
    dispatch_status alertdispatchstatus NOT NULL DEFAULT 'pending',
    telegram_message_id INTEGER,
    acknowledged_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (daily_task_id, alert_kind)
);
CREATE INDEX IF NOT EXISTS ix_daily_task_alerts_daily_task_id ON daily_task_alerts(daily_task_id);

CREATE TABLE IF NOT EXISTS pending_inputs (
    id SERIAL PRIMARY KEY,
    workspace_group_id INTEGER NOT NULL REFERENCES workspace_groups(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL,
    kind VARCHAR(64) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_group_id, user_id)
);
CREATE INDEX IF NOT EXISTS ix_pending_inputs_workspace_group_id ON pending_inputs(workspace_group_id);
CREATE INDEX IF NOT EXISTS ix_pending_inputs_user_id ON pending_inputs(user_id);

CREATE TABLE IF NOT EXISTS app_clock (
    singleton BOOLEAN PRIMARY KEY DEFAULT true CHECK (singleton),
    override_now TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO app_clock (singleton) VALUES (true) ON CONFLICT (singleton) DO NOTHING;

-- +goose Down
-- Baseline rollback is intentionally unsupported: it may represent an existing
-- pre-Go database and must not drop production data.
