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

	if _, err := q.UpsertPendingInput(ctx, workspace.ID, participant.UserID, domain.PendingDailyTaskReport, map[string]any{"thread_id": 11, "task_id": firstTask.ID}); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := q.ClaimPendingInput(ctx, workspace.ID, participant.UserID, domain.PendingDailyTaskReport); err != nil || !ok {
		t.Fatalf("first claim ok=%v err=%v", ok, err)
	}
	if _, ok, err := q.ClaimPendingInput(ctx, workspace.ID, participant.UserID, domain.PendingDailyTaskReport); err != nil || ok {
		t.Fatalf("second claim ok=%v err=%v", ok, err)
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
	if len(leaderboard) != 1 || leaderboard[0].CurrentStreak != 0 || leaderboard[0].CompletionRate != 75 {
		t.Fatalf("unexpected routine leaderboard: %+v", leaderboard)
	}

	period := domain.GoalPeriod{
		Key:      "summer-2026",
		Title:    "Лето 2026",
		StartsOn: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		EndsOn:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
	}
	goalSet, err := q.UpsertSeasonalGoalSet(ctx, workspace.ID, participant.ID, participant.UserID, period, "Результат: оффер\nМетрика: 10 откликов")
	if err != nil {
		t.Fatal(err)
	}
	hasGoals, err := q.HasSeasonalGoalSetForParticipant(ctx, workspace.ID, participant.ID, "summer-2026")
	if err != nil || !hasGoals {
		t.Fatalf("has goals=%v err=%v", hasGoals, err)
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
