package goals_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	appgoals "github.com/igor/trackmate/internal/app/goals"
	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/storage/postgres"
	"github.com/igor/trackmate/internal/telegram"
	"github.com/igor/trackmate/internal/testsupport"
)

type retryTelegram struct {
	fakeTelegram
	fail bool
}

func (f *retryTelegram) SendMessage(ctx context.Context, r telegram.SendMessageRequest) (telegram.Message, error) {
	if f.fail {
		f.fail = false
		return telegram.Message{}, &telegram.Error{StatusCode: 400, Description: "fixture rejection"}
	}
	return f.fakeTelegram.SendMessage(ctx, r)
}
func seedGoalRetry(t *testing.T, store *postgres.Store, n int) time.Time {
	t.Helper()
	ctx := context.Background()
	q := store.Queries()
	w, err := q.GetOrCreateWorkspace(ctx, -100999, "Goals", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertTopicBinding(ctx, w.ID, domain.TopicGoals, 40, "Цели"); err != nil {
		t.Fatal(err)
	}
	period := domain.GoalPeriod{Key: "summer-2026", Title: "Лето", StartsOn: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), EndsOn: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)}
	for i := 1; i <= n; i++ {
		p, err := q.RegisterParticipant(ctx, w.ID, int64(i), fmt.Sprint(i), fmt.Sprint(i))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := q.UpsertSeasonalGoalSet(ctx, w.ID, p.ID, p.UserID, period, "goal", nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	return time.Date(2026, 6, 28, 20, 0, 0, 0, time.UTC)
}
func TestFailedWeeklyReviewDoesNotBlockOthersOrExpiredFinal(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	now := seedGoalRetry(t, store, 2)
	ctx := context.Background()
	tg := &retryTelegram{fail: true}
	if err := appgoals.DispatchWeeklyReviews(ctx, store, tg, now); err != nil {
		t.Fatal(err)
	}
	if len(tg.sent) != 1 {
		t.Fatalf("independent weekly send count=%d", len(tg.sent))
	}
	// The failed initial delivery is already beyond its product deadline at season end.
	end := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	if err := appgoals.DispatchWeeklyReviews(ctx, store, tg, end); err != nil {
		t.Fatal(err)
	}
	if len(tg.sent) != 1 {
		t.Fatal("expired weekly review resent")
	}
	tg.fail = true
	if err := appgoals.DispatchFinalReviews(ctx, store, tg, end); err != nil {
		t.Fatal(err)
	}
	if len(tg.sent) != 2 {
		t.Fatal("failed first final blocked independent final")
	}
	if err := appgoals.DispatchFinalReviews(ctx, store, tg, end); err != nil {
		t.Fatal(err)
	}
	if len(tg.sent) != 2 {
		t.Fatal("final retried before deadline")
	}
	if err := appgoals.DispatchFinalReviews(ctx, store, tg, end.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if len(tg.sent) != 3 {
		t.Fatal("due final not retried")
	}
}
func TestDeferredReminderPreservesPriorPrompt(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	now := seedGoalRetry(t, store, 1)
	ctx := context.Background()
	tg := &retryTelegram{}
	if err := appgoals.DispatchWeeklyReviews(ctx, store, tg, now); err != nil {
		t.Fatal(err)
	}
	tg.fail = true
	if err := appgoals.DispatchWeeklyReviews(ctx, store, tg, now.Add(24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if len(tg.deleted) != 0 || len(tg.sent) != 1 {
		t.Fatal("deferred reminder removed existing prompt")
	}
	if err := appgoals.DispatchWeeklyReviews(ctx, store, tg, now.Add(24*time.Hour+time.Minute)); err != nil {
		t.Fatal(err)
	}
	if len(tg.sent) != 2 || len(tg.deleted) != 1 {
		t.Fatal("successful reminder did not replace previous prompt")
	}
}
