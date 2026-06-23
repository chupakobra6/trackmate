package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/igor/trackmate/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (q *Queries) UpsertRoutinePlan(ctx context.Context, workspaceID int64, participantID int64, ownerUserID int64, items []string) (RoutinePlan, error) {
	encoded, err := json.Marshal(items)
	if err != nil {
		return RoutinePlan{}, err
	}
	row := q.db.QueryRow(ctx, `
INSERT INTO routine_plans (workspace_group_id, participant_id, owner_user_id, items, created_at, updated_at)
VALUES ($1, $2, $3, $4::jsonb, now(), now())
ON CONFLICT (workspace_group_id, participant_id) DO UPDATE SET
    owner_user_id = EXCLUDED.owner_user_id,
    items = EXCLUDED.items,
    updated_at = now()
RETURNING id, workspace_group_id, participant_id, owner_user_id, items, created_at, updated_at
`, workspaceID, participantID, ownerUserID, encoded)
	return scanRoutinePlan(row)
}

func (q *Queries) GetRoutinePlan(ctx context.Context, workspaceID int64, participantID int64) (RoutinePlan, bool, error) {
	row := q.db.QueryRow(ctx, `
SELECT id, workspace_group_id, participant_id, owner_user_id, items, created_at, updated_at
FROM routine_plans
WHERE workspace_group_id = $1 AND participant_id = $2
`, workspaceID, participantID)
	plan, err := scanRoutinePlan(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return RoutinePlan{}, false, nil
	}
	return plan, err == nil, err
}

func (q *Queries) ListRoutinePlanContexts(ctx context.Context) ([]RoutinePlanContext, error) {
	rows, err := q.db.Query(ctx, `
SELECT rp.id, rp.workspace_group_id, rp.participant_id, rp.owner_user_id, rp.items, rp.created_at, rp.updated_at,
       wg.id, wg.chat_id, wg.title, wg.timezone, wg.setup_status::text, wg.setup_message_id, wg.created_at, wg.updated_at,
       p.id, p.workspace_group_id, p.user_id, p.username, p.display_name, p.is_active, p.created_at, p.updated_at
FROM routine_plans rp
JOIN workspace_groups wg ON wg.id = rp.workspace_group_id
JOIN participants p ON p.id = rp.participant_id
WHERE p.is_active = true
ORDER BY rp.id ASC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []RoutinePlanContext
	for rows.Next() {
		var item RoutinePlanContext
		var title, username pgtype.Text
		var setupMessageID pgtype.Int4
		var setupStatus string
		var rawItems []byte
		if err := rows.Scan(
			&item.Plan.ID, &item.Plan.WorkspaceGroupID, &item.Plan.ParticipantID, &item.Plan.OwnerUserID, &rawItems, &item.Plan.CreatedAt, &item.Plan.UpdatedAt,
			&item.Workspace.ID, &item.Workspace.ChatID, &title, &item.Workspace.Timezone, &setupStatus, &setupMessageID, &item.Workspace.CreatedAt, &item.Workspace.UpdatedAt,
			&item.Participant.ID, &item.Participant.WorkspaceGroupID, &item.Participant.UserID, &username, &item.Participant.DisplayName, &item.Participant.IsActive, &item.Participant.CreatedAt, &item.Participant.UpdatedAt,
		); err != nil {
			return nil, err
		}
		item.Plan.Items = decodeStringSlice(rawItems)
		item.Workspace.Title = textFromPg(title)
		item.Workspace.SetupStatus = domain.GroupSetupStatus(setupStatus)
		item.Workspace.SetupMessageID = int64FromPgInt4(setupMessageID)
		item.Participant.Username = textFromPg(username)
		result = append(result, item)
	}
	return result, rows.Err()
}

func (q *Queries) GetOrCreateRoutineCheckin(ctx context.Context, plan RoutinePlan, checkinDate time.Time) (RoutineCheckin, error) {
	row := q.db.QueryRow(ctx, `
INSERT INTO routine_checkins (
    workspace_group_id, participant_id, owner_user_id, checkin_date, created_at, updated_at
)
VALUES ($1, $2, $3, $4::date, now(), now())
ON CONFLICT (workspace_group_id, participant_id, checkin_date) DO UPDATE SET
    updated_at = routine_checkins.updated_at
RETURNING id, workspace_group_id, participant_id, owner_user_id, checkin_date,
          card_message_id, card_message_thread_id, reflection_text, created_at, updated_at, completed_at
`, plan.WorkspaceGroupID, plan.ParticipantID, plan.OwnerUserID, checkinDate)
	checkin, err := scanRoutineCheckin(row)
	if err != nil {
		return RoutineCheckin{}, err
	}
	for index, text := range plan.Items {
		if _, err := q.db.Exec(ctx, `
INSERT INTO routine_checkin_items (routine_checkin_id, item_index, text, created_at, updated_at)
VALUES ($1, $2, $3, now(), now())
ON CONFLICT (routine_checkin_id, item_index) DO NOTHING
`, checkin.ID, index, text); err != nil {
			return RoutineCheckin{}, err
		}
	}
	checkin, _, err = q.GetRoutineCheckin(ctx, checkin.ID)
	return checkin, err
}

func (q *Queries) GetRoutineCheckin(ctx context.Context, checkinID int64) (RoutineCheckin, bool, error) {
	row := q.db.QueryRow(ctx, `
SELECT id, workspace_group_id, participant_id, owner_user_id, checkin_date,
       card_message_id, card_message_thread_id, reflection_text, created_at, updated_at, completed_at
FROM routine_checkins
WHERE id = $1
`, checkinID)
	checkin, err := scanRoutineCheckin(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return RoutineCheckin{}, false, nil
	}
	if err != nil {
		return RoutineCheckin{}, false, err
	}
	items, err := q.ListRoutineCheckinItems(ctx, checkin.ID)
	if err != nil {
		return RoutineCheckin{}, false, err
	}
	checkin.Items = items
	return checkin, true, nil
}

func (q *Queries) GetRoutineCheckinForDate(ctx context.Context, workspaceID int64, participantID int64, checkinDate time.Time) (RoutineCheckin, bool, error) {
	row := q.db.QueryRow(ctx, `
SELECT id, workspace_group_id, participant_id, owner_user_id, checkin_date,
       card_message_id, card_message_thread_id, reflection_text, created_at, updated_at, completed_at
FROM routine_checkins
WHERE workspace_group_id = $1 AND participant_id = $2 AND checkin_date = $3::date
`, workspaceID, participantID, checkinDate)
	checkin, err := scanRoutineCheckin(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return RoutineCheckin{}, false, nil
	}
	if err != nil {
		return RoutineCheckin{}, false, err
	}
	items, err := q.ListRoutineCheckinItems(ctx, checkin.ID)
	if err != nil {
		return RoutineCheckin{}, false, err
	}
	checkin.Items = items
	return checkin, true, nil
}

func (q *Queries) ListRoutineCheckinItems(ctx context.Context, checkinID int64) ([]RoutineCheckinItem, error) {
	rows, err := q.db.Query(ctx, `
SELECT id, routine_checkin_id, item_index, text, status::text, reason_text, created_at, updated_at
FROM routine_checkin_items
WHERE routine_checkin_id = $1
ORDER BY item_index ASC
`, checkinID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []RoutineCheckinItem
	for rows.Next() {
		item, err := scanRoutineCheckinItemRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (q *Queries) SetRoutineCheckinCardMessageID(ctx context.Context, checkinID int64, messageID int64, threadID int64) error {
	_, err := q.db.Exec(ctx, `
UPDATE routine_checkins
SET card_message_id = $2,
    card_message_thread_id = $3,
    updated_at = now()
WHERE id = $1
`, checkinID, messageID, threadID)
	return err
}

func (q *Queries) SetRoutineCheckinItemStatus(ctx context.Context, checkinID int64, ownerUserID int64, itemIndex int, status domain.RoutineItemStatus, reason *string) (RoutineCheckin, bool, error) {
	if !status.IsValid() {
		return RoutineCheckin{}, false, nil
	}
	var reasonValue any
	if reason != nil {
		reasonValue = *reason
	}
	tag, err := q.db.Exec(ctx, `
UPDATE routine_checkin_items rci
SET status = $4::routineitemstatus,
    reason_text = $5,
    updated_at = now()
FROM routine_checkins rc
WHERE rci.routine_checkin_id = rc.id
  AND rc.id = $1
  AND rc.owner_user_id = $2
  AND rci.item_index = $3
  AND rc.completed_at IS NULL
`, checkinID, ownerUserID, itemIndex, string(status), reasonValue)
	if err != nil || tag.RowsAffected() == 0 {
		return RoutineCheckin{}, false, err
	}
	checkin, found, err := q.GetRoutineCheckin(ctx, checkinID)
	return checkin, found, err
}

func (q *Queries) CompleteRoutineCheckin(ctx context.Context, checkinID int64, ownerUserID int64, reflectionHTML string) (RoutineCheckin, bool, error) {
	var missing int
	if err := q.db.QueryRow(ctx, `
SELECT count(*)
FROM routine_checkin_items rci
JOIN routine_checkins rc ON rc.id = rci.routine_checkin_id
WHERE rc.id = $1 AND rc.owner_user_id = $2 AND rci.status IS NULL
`, checkinID, ownerUserID).Scan(&missing); err != nil {
		return RoutineCheckin{}, false, err
	}
	if missing > 0 {
		return RoutineCheckin{}, false, nil
	}
	tag, err := q.db.Exec(ctx, `
UPDATE routine_checkins
SET reflection_text = $3,
    completed_at = COALESCE(completed_at, now()),
    updated_at = now()
WHERE id = $1 AND owner_user_id = $2
`, checkinID, ownerUserID, reflectionHTML)
	if err != nil || tag.RowsAffected() == 0 {
		return RoutineCheckin{}, false, err
	}
	checkin, found, err := q.GetRoutineCheckin(ctx, checkinID)
	return checkin, found, err
}

func (q *Queries) GetRoutineLeaderboard(ctx context.Context, workspaceID int64, nowUTC time.Time) ([]RoutineLeaderboardEntry, error) {
	rows, err := q.db.Query(ctx, `
SELECT rc.id, rc.workspace_group_id, rc.participant_id, rc.owner_user_id, rc.checkin_date,
       rc.card_message_id, rc.card_message_thread_id, rc.reflection_text, rc.created_at, rc.updated_at, rc.completed_at,
       p.id, p.workspace_group_id, p.user_id, p.username, p.display_name, p.is_active, p.created_at, p.updated_at
FROM routine_checkins rc
JOIN participants p ON p.id = rc.participant_id
WHERE rc.workspace_group_id = $1
ORDER BY p.id ASC, rc.checkin_date ASC
`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type participantCheckins struct {
		participant Participant
		checkins    []RoutineCheckin
	}
	byParticipant := map[int64]*participantCheckins{}
	for rows.Next() {
		var checkin RoutineCheckin
		var participant Participant
		var cardID, cardThreadID pgtype.Int4
		var reflection pgtype.Text
		var completedAt pgtype.Timestamptz
		var username pgtype.Text
		if err := rows.Scan(
			&checkin.ID, &checkin.WorkspaceGroupID, &checkin.ParticipantID, &checkin.OwnerUserID, &checkin.CheckinDate,
			&cardID, &cardThreadID, &reflection, &checkin.CreatedAt, &checkin.UpdatedAt, &completedAt,
			&participant.ID, &participant.WorkspaceGroupID, &participant.UserID, &username, &participant.DisplayName, &participant.IsActive, &participant.CreatedAt, &participant.UpdatedAt,
		); err != nil {
			return nil, err
		}
		checkin.CardMessageID = int64FromPgInt4(cardID)
		checkin.CardMessageThreadID = int64FromPgInt4(cardThreadID)
		checkin.ReflectionText = textFromPg(reflection)
		checkin.CompletedAt = timeFromPg(completedAt)
		items, err := q.ListRoutineCheckinItems(ctx, checkin.ID)
		if err != nil {
			return nil, err
		}
		checkin.Items = items
		participant.Username = textFromPg(username)
		if byParticipant[participant.ID] == nil {
			byParticipant[participant.ID] = &participantCheckins{participant: participant}
		}
		byParticipant[participant.ID].checkins = append(byParticipant[participant.ID].checkins, checkin)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	entries := make([]RoutineLeaderboardEntry, 0, len(byParticipant))
	for _, item := range byParticipant {
		entry := RoutineLeaderboardEntry{Participant: item.participant}
		entry.CurrentStreak, entry.MaxStreak = routineStreaks(item.checkins)
		entry.CompletionRate = routineCompletionRate(item.checkins, nowUTC)
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].CurrentStreak != entries[j].CurrentStreak {
			return entries[i].CurrentStreak > entries[j].CurrentStreak
		}
		if entries[i].CompletionRate != entries[j].CompletionRate {
			return entries[i].CompletionRate > entries[j].CompletionRate
		}
		if entries[i].MaxStreak != entries[j].MaxStreak {
			return entries[i].MaxStreak > entries[j].MaxStreak
		}
		return entries[i].Participant.DisplayName < entries[j].Participant.DisplayName
	})
	return entries, nil
}

func routineStreaks(checkins []RoutineCheckin) (int, int) {
	if len(checkins) == 0 {
		return 0, 0
	}
	sort.Slice(checkins, func(i, j int) bool {
		return checkins[i].CheckinDate.Before(checkins[j].CheckinDate)
	})
	maxStreak := 0
	currentRun := 0
	var previous time.Time
	for _, checkin := range checkins {
		if !routineCheckinAllDone(checkin) {
			currentRun = 0
			previous = checkin.CheckinDate
			continue
		}
		if currentRun > 0 && checkin.CheckinDate.Sub(previous) <= 24*time.Hour {
			currentRun++
		} else {
			currentRun = 1
		}
		if currentRun > maxStreak {
			maxStreak = currentRun
		}
		previous = checkin.CheckinDate
	}
	currentStreak := 0
	for i := len(checkins) - 1; i >= 0; i-- {
		if !routineCheckinAllDone(checkins[i]) {
			break
		}
		if currentStreak == 0 {
			currentStreak = 1
			continue
		}
		next := checkins[i+1].CheckinDate
		if next.Sub(checkins[i].CheckinDate) > 24*time.Hour {
			break
		}
		currentStreak++
	}
	return currentStreak, maxStreak
}

func routineCompletionRate(checkins []RoutineCheckin, nowUTC time.Time) float64 {
	if len(checkins) == 0 {
		return 0
	}
	from := time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -6)
	total := 0.0
	count := 0
	for _, checkin := range checkins {
		if checkin.CheckinDate.Before(from) {
			continue
		}
		total += routineCheckinScore(checkin)
		count++
	}
	if count == 0 {
		return 0
	}
	return total / float64(count) * 100
}

func routineCheckinAllDone(checkin RoutineCheckin) bool {
	if len(checkin.Items) == 0 {
		return false
	}
	for _, item := range checkin.Items {
		if item.Status == nil || *item.Status != domain.RoutineItemDone {
			return false
		}
	}
	return true
}

func routineCheckinScore(checkin RoutineCheckin) float64 {
	if len(checkin.Items) == 0 {
		return 0
	}
	total := 0.0
	for _, item := range checkin.Items {
		if item.Status != nil {
			total += domain.RoutineScore(*item.Status)
		}
	}
	return total / float64(len(checkin.Items))
}

func scanRoutinePlan(row pgx.Row) (RoutinePlan, error) {
	var plan RoutinePlan
	var raw []byte
	if err := row.Scan(&plan.ID, &plan.WorkspaceGroupID, &plan.ParticipantID, &plan.OwnerUserID, &raw, &plan.CreatedAt, &plan.UpdatedAt); err != nil {
		return RoutinePlan{}, err
	}
	plan.Items = decodeStringSlice(raw)
	return plan, nil
}

func scanRoutineCheckin(row pgx.Row) (RoutineCheckin, error) {
	var checkin RoutineCheckin
	var cardID, cardThreadID pgtype.Int4
	var reflection pgtype.Text
	var completedAt pgtype.Timestamptz
	if err := row.Scan(
		&checkin.ID, &checkin.WorkspaceGroupID, &checkin.ParticipantID, &checkin.OwnerUserID, &checkin.CheckinDate,
		&cardID, &cardThreadID, &reflection, &checkin.CreatedAt, &checkin.UpdatedAt, &completedAt,
	); err != nil {
		return RoutineCheckin{}, err
	}
	checkin.CardMessageID = int64FromPgInt4(cardID)
	checkin.CardMessageThreadID = int64FromPgInt4(cardThreadID)
	checkin.ReflectionText = textFromPg(reflection)
	checkin.CompletedAt = timeFromPg(completedAt)
	return checkin, nil
}

func scanRoutineCheckinItemRows(rows pgx.Rows) (RoutineCheckinItem, error) {
	var item RoutineCheckinItem
	var status, reason pgtype.Text
	if err := rows.Scan(&item.ID, &item.RoutineCheckinID, &item.ItemIndex, &item.Text, &status, &reason, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return RoutineCheckinItem{}, err
	}
	item.Status = routineStatusFromPg(status)
	item.ReasonText = textFromPg(reason)
	return item, nil
}
