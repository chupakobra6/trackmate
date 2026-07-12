package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/testsupport"
)

func TestDailySummaryDoesNotBlockTomorrowTaskAndPublishesOwnEvent(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	ctx := context.Background()
	q := store.Queries()

	workspace, err := q.GetOrCreateWorkspace(ctx, -1001234567891, "Group", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertTopicBinding(ctx, workspace.ID, domain.TopicToday, 10, "Сегодня"); err != nil {
		t.Fatal(err)
	}
	participant, err := q.RegisterParticipant(ctx, workspace.ID, 42, "igor", "Игорь")
	if err != nil {
		t.Fatal(err)
	}

	summaryDate := time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC)
	summary, created, err := q.CreateDailySummary(ctx, workspace.ID, participant.ID, participant.UserID, summaryDate)
	if err != nil || !created {
		t.Fatalf("summary created=%v err=%v", created, err)
	}
	if summary.Kind != domain.DailyEntrySummary || summary.Text != "" {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if err := q.SetDailyTaskCardMessageID(ctx, summary.ID, 300); err != nil {
		t.Fatal(err)
	}

	again, created, err := q.CreateDailySummary(ctx, workspace.ID, participant.ID, participant.UserID, summaryDate)
	if err != nil || created || again.ID != summary.ID {
		t.Fatalf("same-day summary uniqueness failed: created=%v summary=%+v err=%v", created, again, err)
	}
	if _, found, err := q.GetOpenTask(ctx, workspace.ID, participant.ID); err != nil || found {
		t.Fatalf("open summary must not block a regular task: found=%v err=%v", found, err)
	}

	if _, created, err := q.CreateDailyTask(ctx, workspace.ID, participant.ID, participant.UserID, summaryDate.AddDate(0, 0, 1), "Задача следующего дня", 301, 10); err != nil || !created {
		t.Fatalf("tomorrow task created=%v err=%v", created, err)
	}
	submitted, err := q.SubmitTaskReport(ctx, summary.ID, participant.UserID, domain.DailyTaskPartial, "Сделал главное.", "Игорь", 302, 10)
	if err != nil || !submitted {
		t.Fatalf("summary submitted=%v err=%v", submitted, err)
	}
	events, err := q.ListPendingProgressEvents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].EventType != domain.ProgressDailySummaryClosed {
		t.Fatalf("summary should create its own progress event: %+v", events)
	}
	if _, hasTask := events[0].Payload["task_html"]; hasTask {
		t.Fatalf("summary progress should not use task payload: %+v", events[0].Payload)
	}
}
