package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/storage/postgres"
	"github.com/igor/trackmate/internal/testsupport"
)

func TestWorkerLeaseOwnsAndReleasesOneDatabaseSession(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	acquireCtx, cancelAcquire := context.WithCancel(context.Background())
	lease, acquired, err := store.TryAcquireWorkerLease(acquireCtx)
	if err != nil || !acquired {
		t.Fatalf("worker lease acquired=%v err=%v", acquired, err)
	}
	cancelAcquire()
	defer func() { _ = lease.Release() }()

	competitor, err := store.Pool().Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer competitor.Release()
	var competitorAcquired bool
	if err := competitor.QueryRow(context.Background(), `SELECT pg_try_advisory_lock($1::integer, current_schema()::regnamespace::oid::integer)`, postgres.WorkerLockKey).Scan(&competitorAcquired); err != nil {
		t.Fatal(err)
	}
	if competitorAcquired {
		_, _ = competitor.Exec(context.Background(), `SELECT pg_advisory_unlock($1::integer, current_schema()::regnamespace::oid::integer)`, postgres.WorkerLockKey)
		t.Fatal("competing database session acquired the worker lock")
	}

	if err := lease.Release(); err != nil {
		t.Fatal(err)
	}
	if err := lease.Release(); err != nil {
		t.Fatalf("second release must be idempotent: %v", err)
	}
	if err := competitor.QueryRow(context.Background(), `SELECT pg_try_advisory_lock($1::integer, current_schema()::regnamespace::oid::integer)`, postgres.WorkerLockKey).Scan(&competitorAcquired); err != nil {
		t.Fatal(err)
	}
	if !competitorAcquired {
		t.Fatal("worker lock remained held after lease release")
	}
	if _, err := competitor.Exec(context.Background(), `SELECT pg_advisory_unlock($1::integer, current_schema()::regnamespace::oid::integer)`, postgres.WorkerLockKey); err != nil {
		t.Fatal(err)
	}
}

func TestDeliveryClaimsRecoverAfterLeaseExpiry(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	ctx := context.Background()
	q := store.Queries()
	workspace, err := q.GetOrCreateWorkspace(ctx, -1001234567001, "Delivery", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	participant, err := q.RegisterParticipant(ctx, workspace.ID, 901, "delivery", "Delivery")
	if err != nil {
		t.Fatal(err)
	}
	task, created, err := q.CreateDailyTask(ctx, workspace.ID, participant.ID, participant.UserID, time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC), "Task", 1001, 10)
	if err != nil || !created {
		t.Fatalf("task created=%v err=%v", created, err)
	}

	alert, err := q.GetOrCreateAlert(ctx, task.ID, domain.AlertDayClosedPendingReport)
	if err != nil {
		t.Fatal(err)
	}
	claimedAlert, ok, err := q.ClaimPendingAlert(ctx)
	if err != nil || !ok || claimedAlert.ID != alert.ID {
		t.Fatalf("first alert claim=%+v ok=%v err=%v", claimedAlert, ok, err)
	}
	if _, ok, err := q.ClaimPendingAlert(ctx); err != nil || ok {
		t.Fatalf("fresh alert lease claimable=%v err=%v", ok, err)
	}
	if _, err := store.Pool().Exec(ctx, `UPDATE daily_task_alerts SET dispatch_started_at = now() - interval '6 minutes' WHERE id = $1`, alert.ID); err != nil {
		t.Fatal(err)
	}
	claimedAlert, ok, err = q.ClaimPendingAlert(ctx)
	if err != nil || !ok || claimedAlert.ID != alert.ID {
		t.Fatalf("expired alert claim=%+v ok=%v err=%v", claimedAlert, ok, err)
	}
	if err := q.MarkAlertSent(ctx, alert.ID, 2001); err != nil {
		t.Fatal(err)
	}
	if err := q.MarkAlertSent(ctx, alert.ID, 2002); !errors.Is(err, postgres.ErrDeliveryClaimLost) {
		t.Fatalf("second alert completion error=%v, want ErrDeliveryClaimLost", err)
	}

	event, err := q.CreateProgressEvent(ctx, workspace.ID, domain.ProgressDailyTaskClosed, map[string]any{"status": "done"}, &participant.ID, &task.ID)
	if err != nil {
		t.Fatal(err)
	}
	claimedEvent, ok, err := q.ClaimProgressEvent(ctx)
	if err != nil || !ok || claimedEvent.ID != event.ID {
		t.Fatalf("first progress claim=%+v ok=%v err=%v", claimedEvent, ok, err)
	}
	if _, ok, err := q.ClaimProgressEvent(ctx); err != nil || ok {
		t.Fatalf("fresh progress lease claimable=%v err=%v", ok, err)
	}
	if _, err := store.Pool().Exec(ctx, `UPDATE progress_events SET publish_started_at = NULL WHERE id = $1`, event.ID); err != nil {
		t.Fatal(err)
	}
	claimedEvent, ok, err = q.ClaimProgressEvent(ctx)
	if err != nil || !ok || claimedEvent.ID != event.ID {
		t.Fatalf("legacy progress claim=%+v ok=%v err=%v", claimedEvent, ok, err)
	}
	if err := q.MarkProgressEventPublished(ctx, event.ID, 2003, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := q.MarkProgressEventPublished(ctx, event.ID, 2004, time.Now().UTC()); !errors.Is(err, postgres.ErrDeliveryClaimLost) {
		t.Fatalf("second progress completion error=%v, want ErrDeliveryClaimLost", err)
	}
}
