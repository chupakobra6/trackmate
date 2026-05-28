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
	participant, err := q.RegisterParticipant(ctx, workspace.ID, 42, "igor", "Igor")
	if err != nil {
		t.Fatal(err)
	}

	taskDate := time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC)
	firstTask, created, err := q.CreateDailyTask(ctx, workspace.ID, participant.ID, participant.UserID, taskDate, "Task")
	if err != nil || !created {
		t.Fatalf("first task created=%v err=%v", created, err)
	}
	if err := q.SetDailyTaskCardMessageID(ctx, firstTask.ID, 100); err != nil {
		t.Fatal(err)
	}
	secondTask, created, err := q.CreateDailyTask(ctx, workspace.ID, participant.ID, participant.UserID, taskDate, "Task 2")
	if err != nil {
		t.Fatal(err)
	}
	if created || secondTask.ID != firstTask.ID {
		t.Fatalf("expected uniqueness to return existing task, created=%v second=%d first=%d", created, secondTask.ID, firstTask.ID)
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
	if _, ok, err := q.ClaimProgressEvent(ctx); err != nil || !ok {
		t.Fatalf("progress claim ok=%v err=%v", ok, err)
	}
	if _, ok, err := q.ClaimProgressEvent(ctx); err != nil || ok {
		t.Fatalf("second progress claim ok=%v err=%v", ok, err)
	}

	assertNoTable(t, store, "material_batches")
	assertNoTable(t, store, "material_items")
	assertNoTable(t, store, "material_participant_progresses")
	assertEnumLabels(t, store, "topickey", []string{"today", "progress"})
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
	if reset.DeletedTasks != 1 || reset.DeletedAlerts != 1 || reset.DeletedPending != 0 || reset.DeletedProgress != 1 || reset.ResetSetup != 1 {
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
	} else if len(bindings) != 2 {
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
