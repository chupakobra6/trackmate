package worker_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/storage/postgres"
	"github.com/igor/trackmate/internal/telegram"
	"github.com/igor/trackmate/internal/testsupport"
	"github.com/igor/trackmate/internal/worker"
)

type rejectingTelegram struct {
	fakeTelegram
	reject   func(telegram.SendMessageRequest) error
	attempts int
}

func (f *rejectingTelegram) SendMessage(ctx context.Context, request telegram.SendMessageRequest) (telegram.Message, error) {
	f.attempts++
	if f.reject != nil {
		if err := f.reject(request); err != nil {
			return telegram.Message{}, err
		}
	}
	return f.fakeTelegram.SendMessage(ctx, request)
}

func seedRetryTick(t *testing.T, store *postgres.Store) time.Time {
	t.Helper()
	ctx := context.Background()
	q := store.Queries()
	now := time.Date(2026, 6, 29, 9, 0, 0, 0, time.UTC)
	w, err := q.GetOrCreateWorkspace(ctx, -100123123, "retry", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	for topic, id := range map[domain.TopicKey]int64{domain.TopicRoutine: 30, domain.TopicToday: 10, domain.TopicProgress: 20, domain.TopicGoals: 40} {
		if _, err := q.UpsertTopicBinding(ctx, w.ID, topic, id, string(topic)); err != nil {
			t.Fatal(err)
		}
	}
	for i := int64(1); i <= 2; i++ {
		p, err := q.RegisterParticipant(ctx, w.ID, i, fmt.Sprint(i), fmt.Sprintf("Person%d", i))
		if err != nil {
			t.Fatal(err)
		}
		plan, err := q.UpsertRoutinePlan(ctx, w.ID, p.ID, p.UserID, []string{"habit"}, 0, 0)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := store.Pool().Exec(ctx, `UPDATE routine_plans SET created_at=$1 WHERE id=$2`, now.Add(-48*time.Hour), plan.ID); err != nil {
			t.Fatal(err)
		}
		if i == 2 {
			if _, _, err := q.CreateDailyTask(ctx, w.ID, p.ID, p.UserID, now.AddDate(0, 0, -2), "task", 0, 10); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := q.SetClockOverride(ctx, &now); err != nil {
		t.Fatal(err)
	}
	return now
}

func TestTickLocalFailurePersistsAndIndependentWorkContinues(t *testing.T) {
	store, url := testsupport.OpenMigratedStore(t)
	now := seedRetryTick(t, store)
	ctx := context.Background()
	failed := false
	tg := &rejectingTelegram{}
	tg.reject = func(r telegram.SendMessageRequest) error {
		if r.MessageThreadID == 30 && !failed {
			failed = true
			return &telegram.Error{StatusCode: 400, Description: "fixture rejected card"}
		}
		return nil
	}
	runner := &worker.Runner{Store: store, TG: tg}
	if err := runner.Tick(ctx, now); err != nil {
		t.Fatal(err)
	}
	if !tg.hasThread(30) || !tg.hasThread(10) || !tg.hasThread(20) {
		t.Fatalf("independent card/stages did not complete: %+v", tg.sent)
	}
	count := len(tg.sent)
	attempts := tg.attempts
	reopened, err := postgres.Open(ctx, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	runner = &worker.Runner{Store: reopened, TG: tg}
	if err := runner.Tick(ctx, now.Add(30*time.Second)); err != nil {
		t.Fatal(err)
	}
	if len(tg.sent) != count || tg.attempts != attempts {
		t.Fatal("early retry or repeated successful send after reopening store")
	}
	now = now.Add(time.Minute)
	if err := reopened.Queries().SetClockOverride(ctx, &now); err != nil {
		t.Fatal(err)
	}
	if err := runner.Tick(ctx, now); err != nil {
		t.Fatal(err)
	}
	if len(tg.sent) != count+1 {
		t.Fatalf("want only deferred card, got %d new sends", len(tg.sent)-count)
	}
	if err := runner.Tick(ctx, now); err != nil {
		t.Fatal(err)
	}
	if len(tg.sent) != count+1 {
		t.Fatal("confirmed success repeated")
	}
	var retries int
	if err := reopened.Pool().QueryRow(ctx, `SELECT count(*) FROM worker_delivery_retries`).Scan(&retries); err != nil || retries != 0 {
		t.Fatalf("retries=%d err=%v", retries, err)
	}
}

func TestTickExhaustedDeliveryStopsAtBudget(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	now := seedRetryTick(t, store)
	ctx := context.Background()
	tg := &rejectingTelegram{reject: func(r telegram.SendMessageRequest) error {
		if r.MessageThreadID == 30 {
			return &telegram.Error{StatusCode: 403, Description: "forbidden fixture"}
		}
		return nil
	}}
	runner := &worker.Runner{Store: store, TG: tg}
	for i := 0; i < 9; i++ {
		if err := store.Queries().SetClockOverride(ctx, &now); err != nil {
			t.Fatal(err)
		}
		if err := runner.Tick(ctx, now); err != nil {
			t.Fatal(err)
		}
		now = now.Add(time.Duration(1<<min(i, 6)) * time.Minute)
	}
	var exhausted int
	if err := store.Pool().QueryRow(ctx, `SELECT count(*) FROM worker_delivery_retries WHERE attempts=8 AND exhausted`).Scan(&exhausted); err != nil || exhausted != 2 {
		t.Fatalf("exhausted=%d err=%v", exhausted, err)
	}
	before := tg.attempts
	if err := runner.Tick(ctx, now); err != nil {
		t.Fatal(err)
	}
	if tg.attempts != before {
		t.Fatal("exhausted operation sent again")
	}
}

func TestTickSurfacesGlobalFailureAndCancellation(t *testing.T) {
	for _, failure := range []*telegram.Error{{StatusCode: 503, Description: "service outage"}, {StatusCode: 0, Description: "unknown send outcome", Ambiguous: true}} {
		t.Run(fmt.Sprint(failure.StatusCode), func(t *testing.T) {
			store, _ := testsupport.OpenMigratedStore(t)
			now := seedRetryTick(t, store)
			ctx := context.Background()
			tg := &rejectingTelegram{reject: func(telegram.SendMessageRequest) error { return failure }}
			runner := &worker.Runner{Store: store, TG: tg}
			if err := runner.Tick(ctx, now); !errors.Is(err, failure) {
				t.Fatalf("global failure hidden: %v", err)
			}
			if tg.attempts != 1 {
				t.Fatalf("service failure sent %d requests", tg.attempts)
			}
			var attempts int
			var exhausted bool
			if err := store.Pool().QueryRow(ctx, `SELECT attempts,exhausted FROM worker_delivery_retries`).Scan(&attempts, &exhausted); err != nil || attempts != 1 || exhausted != failure.Ambiguous {
				t.Fatalf("retry state attempts=%d exhausted=%v err=%v", attempts, exhausted, err)
			}
			canceled, cancel := context.WithCancel(ctx)
			cancel()
			if err := runner.Tick(canceled, now); !errors.Is(err, context.Canceled) {
				t.Fatalf("cancellation hidden: %v", err)
			}
			store.Close()
			if err := runner.Tick(ctx, now); err == nil {
				t.Fatal("closed database hidden")
			}
		})
	}
}

func TestTickCancellationAfterPossibleSendPersistsSuspension(t *testing.T) {
	store, url := testsupport.OpenMigratedStore(t)
	now := seedRetryTick(t, store)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tg := &rejectingTelegram{reject: func(telegram.SendMessageRequest) error {
		cancel()
		return &telegram.Error{Ambiguous: true, Description: "response interrupted"}
	}}
	if err := (&worker.Runner{Store: store, TG: tg}).Tick(ctx, now); err == nil {
		t.Fatal("cancellation hidden")
	}
	var exhausted bool
	if err := store.Pool().QueryRow(context.Background(), `SELECT exhausted FROM worker_delivery_retries`).Scan(&exhausted); err != nil || !exhausted {
		t.Fatalf("ambiguous cancellation not suspended: %v %v", exhausted, err)
	}
	restarted, err := postgres.Open(context.Background(), url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	tg.reject = nil
	if err := (&worker.Runner{Store: restarted, TG: tg}).Tick(context.Background(), now); err != nil {
		t.Fatal(err)
	}
	routineSends := 0
	for _, sent := range tg.sent {
		if sent.MessageThreadID == 30 {
			routineSends++
		}
	}
	if routineSends != 1 {
		t.Fatalf("ambiguous card repeated; routine sends=%d", routineSends)
	}
}

func TestTickClaimsDoNotImmediatelyReclaimFailedAlertOrProgress(t *testing.T) {
	for _, thread := range []int64{10, 20} {
		t.Run(fmt.Sprint(thread), func(t *testing.T) {
			store, _ := testsupport.OpenMigratedStore(t)
			now := seedRetryTick(t, store)
			ctx := context.Background()
			q := store.Queries()
			w, _, err := q.GetWorkspaceByChatID(ctx, -100123123)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := q.CreateProgressEvent(ctx, w.ID, domain.ProgressCustomUpdate, map[string]any{"text": "independent"}, nil, nil); err != nil {
				t.Fatal(err)
			}
			first := true
			tg := &rejectingTelegram{reject: func(r telegram.SendMessageRequest) error {
				if r.MessageThreadID == thread && first {
					first = false
					return &telegram.Error{StatusCode: 400, Description: "bad fixture"}
				}
				return nil
			}}
			runner := &worker.Runner{Store: store, TG: tg}
			bounded, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			if err := runner.Tick(bounded, now); err != nil {
				t.Fatal(err)
			}
			var attempts int
			if err := store.Pool().QueryRow(ctx, `SELECT attempts FROM worker_delivery_retries`).Scan(&attempts); err != nil || attempts != 1 {
				t.Fatalf("attempts=%d err=%v", attempts, err)
			}
			if !tg.hasThread(20) {
				t.Fatal("later independent progress failed to advance")
			}
			before := tg.attempts
			if err := runner.Tick(bounded, now); err != nil {
				t.Fatal(err)
			}
			if tg.attempts != before {
				t.Fatal("immediate claim retry")
			}
			now = now.Add(time.Minute)
			if err := q.SetClockOverride(ctx, &now); err != nil {
				t.Fatal(err)
			}
			if err := runner.Tick(bounded, now); err != nil {
				t.Fatal(err)
			}
			if tg.attempts != before+1 {
				t.Fatalf("due retry not exactly once: before %d after %d", before, tg.attempts)
			}
		})
	}
}
