-- +goose NO TRANSACTION
-- +goose Up
ALTER TYPE topickey ADD VALUE IF NOT EXISTS 'routine';
ALTER TYPE topickey ADD VALUE IF NOT EXISTS 'goals';

DO $$ BEGIN
    CREATE TYPE routineitemstatus AS ENUM ('done', 'partial', 'failed');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE goalfinalstatus AS ENUM ('done', 'partial', 'failed');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS routine_plans (
    id SERIAL PRIMARY KEY,
    workspace_group_id INTEGER NOT NULL REFERENCES workspace_groups(id) ON DELETE CASCADE,
    participant_id INTEGER NOT NULL REFERENCES participants(id) ON DELETE CASCADE,
    owner_user_id BIGINT NOT NULL,
    items JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_group_id, participant_id),
    CHECK (jsonb_typeof(items) = 'array')
);
CREATE INDEX IF NOT EXISTS ix_routine_plans_workspace_group_id ON routine_plans(workspace_group_id);
CREATE INDEX IF NOT EXISTS ix_routine_plans_participant_id ON routine_plans(participant_id);
CREATE INDEX IF NOT EXISTS ix_routine_plans_owner_user_id ON routine_plans(owner_user_id);

CREATE TABLE IF NOT EXISTS routine_checkins (
    id SERIAL PRIMARY KEY,
    workspace_group_id INTEGER NOT NULL REFERENCES workspace_groups(id) ON DELETE CASCADE,
    participant_id INTEGER NOT NULL REFERENCES participants(id) ON DELETE CASCADE,
    owner_user_id BIGINT NOT NULL,
    checkin_date DATE NOT NULL,
    card_message_id INTEGER,
    card_message_thread_id INTEGER,
    reflection_text TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    UNIQUE (workspace_group_id, participant_id, checkin_date)
);
CREATE INDEX IF NOT EXISTS ix_routine_checkins_workspace_group_id ON routine_checkins(workspace_group_id);
CREATE INDEX IF NOT EXISTS ix_routine_checkins_participant_id ON routine_checkins(participant_id);
CREATE INDEX IF NOT EXISTS ix_routine_checkins_owner_user_id ON routine_checkins(owner_user_id);
CREATE INDEX IF NOT EXISTS ix_routine_checkins_checkin_date ON routine_checkins(checkin_date);

CREATE TABLE IF NOT EXISTS routine_checkin_items (
    id SERIAL PRIMARY KEY,
    routine_checkin_id INTEGER NOT NULL REFERENCES routine_checkins(id) ON DELETE CASCADE,
    item_index INTEGER NOT NULL,
    text TEXT NOT NULL,
    status routineitemstatus,
    reason_text TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (routine_checkin_id, item_index)
);
CREATE INDEX IF NOT EXISTS ix_routine_checkin_items_checkin_id ON routine_checkin_items(routine_checkin_id);
CREATE INDEX IF NOT EXISTS ix_routine_checkin_items_status ON routine_checkin_items(status);

CREATE TABLE IF NOT EXISTS seasonal_goal_sets (
    id SERIAL PRIMARY KEY,
    workspace_group_id INTEGER NOT NULL REFERENCES workspace_groups(id) ON DELETE CASCADE,
    participant_id INTEGER NOT NULL REFERENCES participants(id) ON DELETE CASCADE,
    owner_user_id BIGINT NOT NULL,
    period_key VARCHAR(64) NOT NULL,
    period_title VARCHAR(255) NOT NULL,
    period_starts_on DATE NOT NULL,
    period_ends_on DATE NOT NULL,
    goals_text TEXT NOT NULL,
    card_message_id INTEGER,
    card_message_thread_id INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_group_id, participant_id, period_key)
);
CREATE INDEX IF NOT EXISTS ix_seasonal_goal_sets_workspace_group_id ON seasonal_goal_sets(workspace_group_id);
CREATE INDEX IF NOT EXISTS ix_seasonal_goal_sets_participant_id ON seasonal_goal_sets(participant_id);
CREATE INDEX IF NOT EXISTS ix_seasonal_goal_sets_owner_user_id ON seasonal_goal_sets(owner_user_id);
CREATE INDEX IF NOT EXISTS ix_seasonal_goal_sets_period_key ON seasonal_goal_sets(period_key);

CREATE TABLE IF NOT EXISTS seasonal_goal_weekly_reviews (
    id SERIAL PRIMARY KEY,
    goal_set_id INTEGER NOT NULL REFERENCES seasonal_goal_sets(id) ON DELETE CASCADE,
    review_week_start DATE NOT NULL,
    prompt_message_id INTEGER,
    prompt_message_thread_id INTEGER,
    response_text TEXT,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    responded_at TIMESTAMPTZ,
    UNIQUE (goal_set_id, review_week_start)
);
CREATE INDEX IF NOT EXISTS ix_goal_weekly_reviews_goal_set_id ON seasonal_goal_weekly_reviews(goal_set_id);
CREATE INDEX IF NOT EXISTS ix_goal_weekly_reviews_week_start ON seasonal_goal_weekly_reviews(review_week_start);

CREATE TABLE IF NOT EXISTS seasonal_goal_final_reviews (
    id SERIAL PRIMARY KEY,
    goal_set_id INTEGER NOT NULL REFERENCES seasonal_goal_sets(id) ON DELETE CASCADE,
    status goalfinalstatus,
    prompt_message_id INTEGER,
    prompt_message_thread_id INTEGER,
    summary_text TEXT,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    UNIQUE (goal_set_id)
);
CREATE INDEX IF NOT EXISTS ix_goal_final_reviews_goal_set_id ON seasonal_goal_final_reviews(goal_set_id);
CREATE INDEX IF NOT EXISTS ix_goal_final_reviews_status ON seasonal_goal_final_reviews(status);

-- +goose Down
DROP TABLE IF EXISTS seasonal_goal_final_reviews;
DROP TABLE IF EXISTS seasonal_goal_weekly_reviews;
DROP TABLE IF EXISTS seasonal_goal_sets;
DROP TABLE IF EXISTS routine_checkin_items;
DROP TABLE IF EXISTS routine_checkins;
DROP TABLE IF EXISTS routine_plans;

DROP TYPE IF EXISTS goalfinalstatus;
DROP TYPE IF EXISTS routineitemstatus;

-- PostgreSQL enum labels are intentionally retained in topickey. Removing enum
-- labels requires rewriting dependent columns and is riskier than leaving the
-- harmless labels in a rollback.
