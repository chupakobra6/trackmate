package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/storage/postgres"
	"github.com/igor/trackmate/internal/testsupport"
)

func TestStorageIntegrationContracts(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	ctx := context.Background()
	q := store.Queries()

	workspace, err := q.GetOrCreateWorkspace(ctx, -1001234567890, "Group", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertTopicBinding(ctx, workspace.ID, domain.TopicToday, 11, "Сегодня"); err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertTopicBinding(ctx, workspace.ID, domain.TopicProgress, 12, "Прогресс"); err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertTopicBinding(ctx, workspace.ID, domain.TopicRoutine, 13, "Рутины"); err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertTopicBinding(ctx, workspace.ID, domain.TopicGoals, 14, "Цели"); err != nil {
		t.Fatal(err)
	}
	participant, err := q.RegisterParticipant(ctx, workspace.ID, 42, "igor", "Igor")
	if err != nil {
		t.Fatal(err)
	}

	taskDate := time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC)
	firstTask, created, err := q.CreateDailyTask(ctx, workspace.ID, participant.ID, participant.UserID, taskDate, "Task", 201, 11)
	if err != nil || !created {
		t.Fatalf("first task created=%v err=%v", created, err)
	}
	if err := q.SetDailyTaskCardMessageID(ctx, firstTask.ID, 100); err != nil {
		t.Fatal(err)
	}
	secondTask, created, err := q.CreateDailyTask(ctx, workspace.ID, participant.ID, participant.UserID, taskDate, "Task 2", 202, 11)
	if err != nil {
		t.Fatal(err)
	}
	if created || secondTask.ID != firstTask.ID {
		t.Fatalf("expected uniqueness to return existing task, created=%v second=%d first=%d", created, secondTask.ID, firstTask.ID)
	}
	if firstTask.TaskMessageID == nil || *firstTask.TaskMessageID != 201 || firstTask.TaskMessageThreadID == nil || *firstTask.TaskMessageThreadID != 11 {
		t.Fatalf("source message was not stored: %+v", firstTask)
	}

	if _, err := q.UpsertPendingInput(ctx, workspace.ID, participant.UserID, 11, domain.PendingDailyTaskReport, map[string]any{"task_id": firstTask.ID}); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := q.ClaimPendingInput(ctx, workspace.ID, participant.UserID, 11, domain.PendingDailyTaskReport); err != nil || !ok {
		t.Fatalf("first claim ok=%v err=%v", ok, err)
	}
	if _, ok, err := q.ClaimPendingInput(ctx, workspace.ID, participant.UserID, 11, domain.PendingDailyTaskReport); err != nil || ok {
		t.Fatalf("second claim ok=%v err=%v", ok, err)
	}
	if _, err := q.UpsertPendingInput(ctx, workspace.ID, participant.UserID, 13, domain.PendingRoutinePlan, map[string]any{"prompt_message_id": 301}); err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertPendingInput(ctx, workspace.ID, participant.UserID, 14, domain.PendingSeasonalGoals, map[string]any{"prompt_message_id": 401}); err != nil {
		t.Fatal(err)
	}
	if pending, found, err := q.GetPendingInput(ctx, workspace.ID, participant.UserID, 13); err != nil || !found || pending.Kind != domain.PendingRoutinePlan || pending.MessageThreadID != 13 {
		t.Fatalf("routine pending found=%v pending=%+v err=%v", found, pending, err)
	}
	if pending, found, err := q.GetPendingInput(ctx, workspace.ID, participant.UserID, 14); err != nil || !found || pending.Kind != domain.PendingSeasonalGoals || pending.MessageThreadID != 14 {
		t.Fatalf("goals pending found=%v pending=%+v err=%v", found, pending, err)
	}
	if err := q.ClearPendingInput(ctx, workspace.ID, participant.UserID, 13); err != nil {
		t.Fatal(err)
	}
	if _, found, err := q.GetPendingInput(ctx, workspace.ID, participant.UserID, 13); err != nil || found {
		t.Fatalf("routine pending after clear found=%v err=%v", found, err)
	}
	if _, found, err := q.GetPendingInput(ctx, workspace.ID, participant.UserID, 14); err != nil || !found {
		t.Fatalf("goals pending should remain found=%v err=%v", found, err)
	}
	if err := q.ClearPendingInput(ctx, workspace.ID, participant.UserID, 14); err != nil {
		t.Fatal(err)
	}

	if _, err := q.GetOrCreateAlert(ctx, firstTask.ID, domain.AlertDayClosedPendingReport); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := q.ClaimPendingAlert(ctx); err != nil || !ok {
		t.Fatalf("alert claim ok=%v err=%v", ok, err)
	}
	if _, ok, err := q.ClaimPendingAlert(ctx); err != nil || ok {
		t.Fatalf("second alert claim ok=%v err=%v", ok, err)
	}

	if _, err := q.CreateProgressEvent(ctx, workspace.ID, domain.ProgressDailyTaskClosed, map[string]any{"status": "done"}, &participant.ID, &firstTask.ID); err != nil {
		t.Fatal(err)
	}
	updatedTask, syncedEvents, ok, err := q.UpdateTaskTextFromSourceMessage(ctx, workspace.ID, participant.UserID, 201, 11, "Updated task")
	if err != nil || !ok {
		t.Fatalf("source edit ok=%v err=%v", ok, err)
	}
	if updatedTask.Text != "Updated task" {
		t.Fatalf("updated task text = %q", updatedTask.Text)
	}
	if len(syncedEvents) != 1 || syncedEvents[0].Payload["task_html"] != "Updated task" {
		t.Fatalf("progress payload was not synced: %+v", syncedEvents)
	}
	if _, ok, err := q.ClaimProgressEvent(ctx); err != nil || !ok {
		t.Fatalf("progress claim ok=%v err=%v", ok, err)
	}
	if _, ok, err := q.ClaimProgressEvent(ctx); err != nil || ok {
		t.Fatalf("second progress claim ok=%v err=%v", ok, err)
	}

	routinePlan, err := q.UpsertRoutinePlan(ctx, workspace.ID, participant.ID, participant.UserID, []string{"зарядка", "йога"})
	if err != nil {
		t.Fatal(err)
	}
	checkin, err := q.GetOrCreateRoutineCheckin(ctx, routinePlan, time.Date(2026, 5, 29, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(checkin.Items) != 2 {
		t.Fatalf("routine items = %d, want 2", len(checkin.Items))
	}
	if _, ok, err := q.SetRoutineCheckinItemStatus(ctx, checkin.ID, participant.UserID, 0, domain.RoutineItemDone, nil); err != nil || !ok {
		t.Fatalf("routine item status ok=%v err=%v", ok, err)
	}
	reason := "не хватило времени"
	if _, ok, err := q.SetRoutineCheckinItemStatus(ctx, checkin.ID, participant.UserID, 1, domain.RoutineItemPartial, &reason); err != nil || !ok {
		t.Fatalf("routine partial ok=%v err=%v", ok, err)
	}
	completed, ok, err := q.CompleteRoutineCheckin(ctx, checkin.ID, participant.UserID, "Что помогло / что мешало / правка")
	if err != nil || !ok {
		t.Fatalf("routine complete ok=%v err=%v", ok, err)
	}
	if completed.CompletedAt == nil || completed.ReflectionText == nil {
		t.Fatalf("routine completion not stored: %+v", completed)
	}
	leaderboard, err := q.GetRoutineLeaderboard(ctx, workspace.ID, time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(leaderboard) != 1 || leaderboard[0].CurrentStreak != 0 || leaderboard[0].CompletionRate != 75 || leaderboard[0].RoutineItemCount != 2 {
		t.Fatalf("unexpected routine leaderboard: %+v", leaderboard)
	}

	period := domain.GoalPeriod{
		Key:      "summer-2026",
		Title:    "Лето 2026",
		StartsOn: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		EndsOn:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
	}
	goalSet, err := q.UpsertSeasonalGoalSet(ctx, workspace.ID, participant.ID, participant.UserID, period, "Результат: предложение о работе\nМетрика: 10 откликов")
	if err != nil {
		t.Fatal(err)
	}
	hasGoals, err := q.HasSeasonalGoalSetForParticipant(ctx, workspace.ID, participant.ID, "summer-2026")
	if err != nil || !hasGoals {
		t.Fatalf("has goals=%v err=%v", hasGoals, err)
	}
	if _, found, err := q.GetGoalNudgeCooldown(ctx, workspace.ID, participant.ID); err != nil || found {
		t.Fatalf("unexpected initial nudge cooldown found=%v err=%v", found, err)
	}
	nudgeAt := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	if err := q.MarkGoalNudgeShown(ctx, workspace.ID, participant.ID, nudgeAt); err != nil {
		t.Fatal(err)
	}
	cooldown, found, err := q.GetGoalNudgeCooldown(ctx, workspace.ID, participant.ID)
	if err != nil || !found || !cooldown.LastShownAt.Equal(nudgeAt) {
		t.Fatalf("nudge cooldown found=%v cooldown=%+v err=%v", found, cooldown, err)
	}
	weekly, err := q.GetOrCreateGoalWeeklyReview(ctx, goalSet.ID, time.Date(2026, 6, 22, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if err := q.SetGoalWeeklyReviewPrompt(ctx, weekly.ID, 701, 14); err != nil {
		t.Fatal(err)
	}
	weekly, ok, err = q.SubmitGoalWeeklyReview(ctx, weekly.ID, participant.UserID, "Сдвинулось: 6 откликов")
	if err != nil || !ok || weekly.ResponseText == nil {
		t.Fatalf("weekly submit ok=%v review=%+v err=%v", ok, weekly, err)
	}
	final, err := q.GetOrCreateGoalFinalReview(ctx, goalSet.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := q.SetGoalFinalReviewPrompt(ctx, final.ID, 702, 14); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := q.SetGoalFinalReviewStatus(ctx, goalSet.ID, participant.UserID, domain.GoalFinalPartial); err != nil || !ok {
		t.Fatalf("final status ok=%v err=%v", ok, err)
	}
	final, ok, err = q.CompleteGoalFinalReview(ctx, goalSet.ID, participant.UserID, "Получилось частично")
	if err != nil || !ok || final.CompletedAt == nil {
		t.Fatalf("final complete ok=%v review=%+v err=%v", ok, final, err)
	}

	assertNoTable(t, store, "material_batches")
	assertNoTable(t, store, "material_items")
	assertNoTable(t, store, "material_participant_progresses")
	assertEnumLabels(t, store, "topickey", []string{"today", "progress", "routine", "goals"})
	assertEnumLabels(t, store, "progresseventtype", []string{"daily_task.closed", "daily_task.auto_failed", "system_alert", "custom_update"})

	if err := q.SetSetupMessageID(ctx, workspace.ID, 777); err != nil {
		t.Fatal(err)
	}
	if err := q.MarkWorkspaceReady(ctx, workspace.ID); err != nil {
		t.Fatal(err)
	}
	reset, err := q.ResetWorkspaceForE2E(ctx, workspace.ChatID)
	if err != nil {
		t.Fatal(err)
	}
	if reset.DeletedTasks != 1 || reset.DeletedAlerts != 1 || reset.DeletedPending != 0 || reset.DeletedProgress != 1 || reset.DeletedRoutines != 1 || reset.DeletedGoals != 1 || reset.ResetSetup != 1 {
		t.Fatalf("unexpected reset result: %+v", reset)
	}
	reloaded, found, err := q.GetWorkspaceByChatID(ctx, workspace.ChatID)
	if err != nil || !found {
		t.Fatalf("workspace after reset found=%v err=%v", found, err)
	}
	if reloaded.SetupStatus != domain.GroupSetupPending || reloaded.SetupMessageID != nil {
		t.Fatalf("setup reset mismatch: status=%s setup_message_id=%v", reloaded.SetupStatus, reloaded.SetupMessageID)
	}
	if bindings, err := q.ListTopicBindings(ctx, workspace.ID); err != nil {
		t.Fatal(err)
	} else if len(bindings) != 4 {
		t.Fatalf("topic bindings should stay intact after reset, got %d", len(bindings))
	}
}

func TestRoutineLeaderboardRanksCompletionRateBeforeStreak(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	ctx := context.Background()
	q := store.Queries()

	workspace, err := q.GetOrCreateWorkspace(ctx, -1001234567891, "Group", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	streakParticipant, err := q.RegisterParticipant(ctx, workspace.ID, 101, "streak", "Streak")
	if err != nil {
		t.Fatal(err)
	}
	rateParticipant, err := q.RegisterParticipant(ctx, workspace.ID, 102, "rate", "Rate")
	if err != nil {
		t.Fatal(err)
	}
	streakPlan, err := q.UpsertRoutinePlan(ctx, workspace.ID, streakParticipant.ID, streakParticipant.UserID, []string{"одно действие"})
	if err != nil {
		t.Fatal(err)
	}
	ratePlan, err := q.UpsertRoutinePlan(ctx, workspace.ID, rateParticipant.ID, rateParticipant.UserID, []string{"зарядка", "работа", "английский", "йога"})
	if err != nil {
		t.Fatal(err)
	}

	start := time.Date(2026, 6, 17, 0, 0, 0, 0, time.UTC)
	for day := 0; day < 7; day++ {
		date := start.AddDate(0, 0, day)
		streakCheckin, err := q.GetOrCreateRoutineCheckin(ctx, streakPlan, date)
		if err != nil {
			t.Fatal(err)
		}
		streakStatus := domain.RoutineItemFailed
		if day >= 4 {
			streakStatus = domain.RoutineItemDone
		}
		if _, ok, err := q.SetRoutineCheckinItemStatus(ctx, streakCheckin.ID, streakParticipant.UserID, 0, streakStatus, nil); err != nil || !ok {
			t.Fatalf("streak item ok=%v err=%v", ok, err)
		}

		rateCheckin, err := q.GetOrCreateRoutineCheckin(ctx, ratePlan, date)
		if err != nil {
			t.Fatal(err)
		}
		for item := 0; item < 3; item++ {
			if _, ok, err := q.SetRoutineCheckinItemStatus(ctx, rateCheckin.ID, rateParticipant.UserID, item, domain.RoutineItemDone, nil); err != nil || !ok {
				t.Fatalf("rate done item ok=%v err=%v", ok, err)
			}
		}
		reason := "не успел"
		if _, ok, err := q.SetRoutineCheckinItemStatus(ctx, rateCheckin.ID, rateParticipant.UserID, 3, domain.RoutineItemPartial, &reason); err != nil || !ok {
			t.Fatalf("rate partial item ok=%v err=%v", ok, err)
		}
	}

	leaderboard, err := q.GetRoutineLeaderboard(ctx, workspace.ID, time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(leaderboard) != 2 {
		t.Fatalf("leaderboard entries = %d, want 2: %+v", len(leaderboard), leaderboard)
	}
	if leaderboard[0].Participant.UserID != rateParticipant.UserID {
		t.Fatalf("completion rate should rank first, got %+v", leaderboard)
	}
	if leaderboard[0].CompletionRate <= leaderboard[1].CompletionRate || leaderboard[1].CurrentStreak != 3 {
		t.Fatalf("unexpected metrics: %+v", leaderboard)
	}
}

func assertNoTable(t *testing.T, store *postgres.Store, name string) {
	t.Helper()
	var exists bool
	err := store.Pool().QueryRow(context.Background(), `
SELECT EXISTS (
    SELECT 1
    FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = $1
)
`, name).Scan(&exists)
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatalf("removed table still exists: %s", name)
	}
}

func assertEnumLabels(t *testing.T, store *postgres.Store, name string, want []string) {
	t.Helper()
	rows, err := store.Pool().Query(context.Background(), `
SELECT e.enumlabel
FROM pg_enum e
JOIN pg_type t ON t.oid = e.enumtypid
JOIN pg_namespace n ON n.oid = t.typnamespace
WHERE t.typname = $1
  AND n.nspname = current_schema()
ORDER BY e.enumsortorder
`, name)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got []string
	for rows.Next() {
		var label string
		if err := rows.Scan(&label); err != nil {
			t.Fatal(err)
		}
		got = append(got, label)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("%s labels got %v want %v", name, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s labels got %v want %v", name, got, want)
		}
	}
}
