package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/igor/trackmate/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (q *Queries) UpsertSeasonalGoalSet(ctx context.Context, workspaceID int64, participantID int64, ownerUserID int64, period domain.GoalPeriod, goalsHTML string, sourceMessageID *int64, sourceMessageThreadID *int64) (SeasonalGoalSet, error) {
	row := q.db.QueryRow(ctx, `
INSERT INTO seasonal_goal_sets (
    workspace_group_id, participant_id, owner_user_id, period_key, period_title,
    period_starts_on, period_ends_on, goals_text, source_message_id, source_message_thread_id, created_at, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6::date, $7::date, $8, $9, $10, now(), now())
ON CONFLICT (workspace_group_id, participant_id, period_key) DO UPDATE SET
    owner_user_id = EXCLUDED.owner_user_id,
    period_title = EXCLUDED.period_title,
    period_starts_on = EXCLUDED.period_starts_on,
    period_ends_on = EXCLUDED.period_ends_on,
    goals_text = EXCLUDED.goals_text,
    source_message_id = EXCLUDED.source_message_id,
    source_message_thread_id = EXCLUDED.source_message_thread_id,
    updated_at = now()
RETURNING id, workspace_group_id, participant_id, owner_user_id, period_key, period_title,
          period_starts_on, period_ends_on, goals_text, card_message_id, card_message_thread_id,
          source_message_id, source_message_thread_id, created_at, updated_at
`, workspaceID, participantID, ownerUserID, period.Key, period.Title, period.StartsOn, period.EndsOn, goalsHTML, sourceMessageID, sourceMessageThreadID)
	return scanSeasonalGoalSet(row)
}

func (q *Queries) GetSeasonalGoalSet(ctx context.Context, goalSetID int64) (SeasonalGoalSet, bool, error) {
	row := q.db.QueryRow(ctx, `
SELECT id, workspace_group_id, participant_id, owner_user_id, period_key, period_title,
       period_starts_on, period_ends_on, goals_text, card_message_id, card_message_thread_id,
       source_message_id, source_message_thread_id, created_at, updated_at
FROM seasonal_goal_sets
WHERE id = $1
`, goalSetID)
	goalSet, err := scanSeasonalGoalSet(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return SeasonalGoalSet{}, false, nil
	}
	return goalSet, err == nil, err
}

func (q *Queries) HasSeasonalGoalSetForParticipant(ctx context.Context, workspaceID int64, participantID int64, periodKey string) (bool, error) {
	var exists bool
	err := q.db.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1
    FROM seasonal_goal_sets
    WHERE workspace_group_id = $1
      AND participant_id = $2
      AND period_key = $3
)
`, workspaceID, participantID, periodKey).Scan(&exists)
	return exists, err
}

func (q *Queries) GetGoalNudgeCooldown(ctx context.Context, workspaceID int64, participantID int64) (GoalNudgeCooldown, bool, error) {
	row := q.db.QueryRow(ctx, `
SELECT workspace_group_id, participant_id, last_shown_at, created_at, updated_at
FROM goal_nudge_cooldowns
WHERE workspace_group_id = $1 AND participant_id = $2
`, workspaceID, participantID)
	var cooldown GoalNudgeCooldown
	if err := row.Scan(&cooldown.WorkspaceGroupID, &cooldown.ParticipantID, &cooldown.LastShownAt, &cooldown.CreatedAt, &cooldown.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return GoalNudgeCooldown{}, false, nil
		}
		return GoalNudgeCooldown{}, false, err
	}
	return cooldown, true, nil
}

func (q *Queries) MarkGoalNudgeShown(ctx context.Context, workspaceID int64, participantID int64, shownAt time.Time) error {
	_, err := q.db.Exec(ctx, `
INSERT INTO goal_nudge_cooldowns (workspace_group_id, participant_id, last_shown_at, created_at, updated_at)
VALUES ($1, $2, $3, now(), now())
ON CONFLICT (workspace_group_id, participant_id) DO UPDATE SET
    last_shown_at = EXCLUDED.last_shown_at,
    updated_at = now()
`, workspaceID, participantID, shownAt.UTC())
	return err
}

func (q *Queries) SetSeasonalGoalCardMessageID(ctx context.Context, goalSetID int64, messageID int64, threadID int64) error {
	_, err := q.db.Exec(ctx, `
UPDATE seasonal_goal_sets
SET card_message_id = $2,
    card_message_thread_id = $3,
    updated_at = now()
WHERE id = $1
`, goalSetID, messageID, threadID)
	return err
}

func (q *Queries) ClearSeasonalGoalCardMessageID(ctx context.Context, goalSetID int64) error {
	_, err := q.db.Exec(ctx, `
UPDATE seasonal_goal_sets
SET card_message_id = NULL,
    card_message_thread_id = NULL,
    updated_at = now()
WHERE id = $1
`, goalSetID)
	return err
}

func (q *Queries) ListSeasonalGoalSetContexts(ctx context.Context) ([]SeasonalGoalSetContext, error) {
	rows, err := q.db.Query(ctx, `
SELECT gs.id, gs.workspace_group_id, gs.participant_id, gs.owner_user_id, gs.period_key, gs.period_title,
       gs.period_starts_on, gs.period_ends_on, gs.goals_text, gs.card_message_id, gs.card_message_thread_id,
       gs.source_message_id, gs.source_message_thread_id, gs.created_at, gs.updated_at,
       wg.id, wg.chat_id, wg.title, wg.timezone, wg.setup_status::text, wg.setup_message_id, wg.created_at, wg.updated_at,
       p.id, p.workspace_group_id, p.user_id, p.username, p.display_name, p.is_active, p.created_at, p.updated_at
FROM seasonal_goal_sets gs
JOIN workspace_groups wg ON wg.id = gs.workspace_group_id
JOIN participants p ON p.id = gs.participant_id
WHERE p.is_active = true
ORDER BY gs.id ASC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []SeasonalGoalSetContext
	for rows.Next() {
		var item SeasonalGoalSetContext
		var goalCard, goalThread, sourceMessage, sourceThread pgtype.Int4
		var title, username pgtype.Text
		var setupMessageID pgtype.Int4
		var setupStatus string
		if err := rows.Scan(
			&item.GoalSet.ID, &item.GoalSet.WorkspaceGroupID, &item.GoalSet.ParticipantID, &item.GoalSet.OwnerUserID, &item.GoalSet.PeriodKey, &item.GoalSet.PeriodTitle,
			&item.GoalSet.PeriodStartsOn, &item.GoalSet.PeriodEndsOn, &item.GoalSet.GoalsText, &goalCard, &goalThread, &sourceMessage, &sourceThread, &item.GoalSet.CreatedAt, &item.GoalSet.UpdatedAt,
			&item.Workspace.ID, &item.Workspace.ChatID, &title, &item.Workspace.Timezone, &setupStatus, &setupMessageID, &item.Workspace.CreatedAt, &item.Workspace.UpdatedAt,
			&item.Participant.ID, &item.Participant.WorkspaceGroupID, &item.Participant.UserID, &username, &item.Participant.DisplayName, &item.Participant.IsActive, &item.Participant.CreatedAt, &item.Participant.UpdatedAt,
		); err != nil {
			return nil, err
		}
		item.GoalSet.CardMessageID = int64FromPgInt4(goalCard)
		item.GoalSet.CardMessageThreadID = int64FromPgInt4(goalThread)
		item.GoalSet.SourceMessageID = int64FromPgInt4(sourceMessage)
		item.GoalSet.SourceMessageThreadID = int64FromPgInt4(sourceThread)
		item.Workspace.Title = textFromPg(title)
		item.Workspace.SetupStatus = domain.GroupSetupStatus(setupStatus)
		item.Workspace.SetupMessageID = int64FromPgInt4(setupMessageID)
		item.Participant.Username = textFromPg(username)
		result = append(result, item)
	}
	return result, rows.Err()
}

func (q *Queries) GetOrCreateGoalWeeklyReview(ctx context.Context, goalSetID int64, weekStart time.Time) (GoalWeeklyReview, error) {
	row := q.db.QueryRow(ctx, `
INSERT INTO seasonal_goal_weekly_reviews (goal_set_id, review_week_start, requested_at)
VALUES ($1, $2::date, now())
ON CONFLICT (goal_set_id, review_week_start) DO UPDATE SET
    requested_at = seasonal_goal_weekly_reviews.requested_at
RETURNING id, goal_set_id, review_week_start, prompt_message_id, prompt_message_thread_id,
          response_text, requested_at, responded_at
`, goalSetID, weekStart)
	return scanGoalWeeklyReview(row)
}

func (q *Queries) SetGoalWeeklyReviewPrompt(ctx context.Context, reviewID int64, messageID int64, threadID int64) error {
	_, err := q.db.Exec(ctx, `
UPDATE seasonal_goal_weekly_reviews
SET prompt_message_id = $2,
    prompt_message_thread_id = $3
WHERE id = $1
`, reviewID, messageID, threadID)
	return err
}

func (q *Queries) SubmitGoalWeeklyReview(ctx context.Context, reviewID int64, ownerUserID int64, responseHTML string) (GoalWeeklyReview, bool, error) {
	row := q.db.QueryRow(ctx, `
UPDATE seasonal_goal_weekly_reviews gwr
SET response_text = $3,
    responded_at = now()
FROM seasonal_goal_sets gs
WHERE gwr.goal_set_id = gs.id
  AND gwr.id = $1
  AND gs.owner_user_id = $2
RETURNING gwr.id, gwr.goal_set_id, gwr.review_week_start, gwr.prompt_message_id, gwr.prompt_message_thread_id,
          gwr.response_text, gwr.requested_at, gwr.responded_at
`, reviewID, ownerUserID, responseHTML)
	review, err := scanGoalWeeklyReview(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return GoalWeeklyReview{}, false, nil
	}
	return review, err == nil, err
}

func (q *Queries) GetOrCreateGoalFinalReview(ctx context.Context, goalSetID int64) (GoalFinalReview, error) {
	row := q.db.QueryRow(ctx, `
INSERT INTO seasonal_goal_final_reviews (goal_set_id, requested_at)
VALUES ($1, now())
ON CONFLICT (goal_set_id) DO UPDATE SET
    requested_at = seasonal_goal_final_reviews.requested_at
RETURNING id, goal_set_id, status::text, prompt_message_id, prompt_message_thread_id,
          summary_text, requested_at, completed_at
`, goalSetID)
	return scanGoalFinalReview(row)
}

func (q *Queries) SetGoalFinalReviewPrompt(ctx context.Context, reviewID int64, messageID int64, threadID int64) error {
	_, err := q.db.Exec(ctx, `
UPDATE seasonal_goal_final_reviews
SET prompt_message_id = $2,
    prompt_message_thread_id = $3
WHERE id = $1
`, reviewID, messageID, threadID)
	return err
}

func (q *Queries) SetGoalFinalReviewStatus(ctx context.Context, goalSetID int64, ownerUserID int64, status domain.GoalFinalStatus) (GoalFinalReview, bool, error) {
	if !status.IsValid() {
		return GoalFinalReview{}, false, nil
	}
	row := q.db.QueryRow(ctx, `
UPDATE seasonal_goal_final_reviews gfr
SET status = $3::goalfinalstatus
FROM seasonal_goal_sets gs
WHERE gfr.goal_set_id = gs.id
  AND gfr.goal_set_id = $1
  AND gs.owner_user_id = $2
  AND gfr.completed_at IS NULL
RETURNING gfr.id, gfr.goal_set_id, gfr.status::text, gfr.prompt_message_id, gfr.prompt_message_thread_id,
          gfr.summary_text, gfr.requested_at, gfr.completed_at
`, goalSetID, ownerUserID, string(status))
	review, err := scanGoalFinalReview(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return GoalFinalReview{}, false, nil
	}
	return review, err == nil, err
}

func (q *Queries) CompleteGoalFinalReview(ctx context.Context, goalSetID int64, ownerUserID int64, summaryHTML string) (GoalFinalReview, bool, error) {
	row := q.db.QueryRow(ctx, `
UPDATE seasonal_goal_final_reviews gfr
SET summary_text = $3,
    completed_at = now()
FROM seasonal_goal_sets gs
WHERE gfr.goal_set_id = gs.id
  AND gfr.goal_set_id = $1
  AND gs.owner_user_id = $2
  AND gfr.status IS NOT NULL
RETURNING gfr.id, gfr.goal_set_id, gfr.status::text, gfr.prompt_message_id, gfr.prompt_message_thread_id,
          gfr.summary_text, gfr.requested_at, gfr.completed_at
`, goalSetID, ownerUserID, summaryHTML)
	review, err := scanGoalFinalReview(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return GoalFinalReview{}, false, nil
	}
	return review, err == nil, err
}

func scanSeasonalGoalSet(row pgx.Row) (SeasonalGoalSet, error) {
	var goalSet SeasonalGoalSet
	var cardID, cardThreadID, sourceID, sourceThreadID pgtype.Int4
	if err := row.Scan(
		&goalSet.ID, &goalSet.WorkspaceGroupID, &goalSet.ParticipantID, &goalSet.OwnerUserID, &goalSet.PeriodKey, &goalSet.PeriodTitle,
		&goalSet.PeriodStartsOn, &goalSet.PeriodEndsOn, &goalSet.GoalsText, &cardID, &cardThreadID, &sourceID, &sourceThreadID, &goalSet.CreatedAt, &goalSet.UpdatedAt,
	); err != nil {
		return SeasonalGoalSet{}, err
	}
	goalSet.CardMessageID = int64FromPgInt4(cardID)
	goalSet.CardMessageThreadID = int64FromPgInt4(cardThreadID)
	goalSet.SourceMessageID = int64FromPgInt4(sourceID)
	goalSet.SourceMessageThreadID = int64FromPgInt4(sourceThreadID)
	return goalSet, nil
}

func scanGoalWeeklyReview(row pgx.Row) (GoalWeeklyReview, error) {
	var review GoalWeeklyReview
	var promptID, promptThreadID pgtype.Int4
	var response pgtype.Text
	var respondedAt pgtype.Timestamptz
	if err := row.Scan(
		&review.ID, &review.GoalSetID, &review.ReviewWeekStart, &promptID, &promptThreadID,
		&response, &review.RequestedAt, &respondedAt,
	); err != nil {
		return GoalWeeklyReview{}, err
	}
	review.PromptMessageID = int64FromPgInt4(promptID)
	review.PromptMessageThreadID = int64FromPgInt4(promptThreadID)
	review.ResponseText = textFromPg(response)
	review.RespondedAt = timeFromPg(respondedAt)
	return review, nil
}

func scanGoalFinalReview(row pgx.Row) (GoalFinalReview, error) {
	var review GoalFinalReview
	var status, summary pgtype.Text
	var promptID, promptThreadID pgtype.Int4
	var completedAt pgtype.Timestamptz
	if err := row.Scan(
		&review.ID, &review.GoalSetID, &status, &promptID, &promptThreadID,
		&summary, &review.RequestedAt, &completedAt,
	); err != nil {
		return GoalFinalReview{}, err
	}
	review.Status = goalFinalStatusFromPg(status)
	review.PromptMessageID = int64FromPgInt4(promptID)
	review.PromptMessageThreadID = int64FromPgInt4(promptThreadID)
	review.SummaryText = textFromPg(summary)
	review.CompletedAt = timeFromPg(completedAt)
	return review, nil
}
