package goals_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	appgoals "github.com/igor/trackmate/internal/app/goals"
	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/testsupport"
)

func TestMaybeNudgeUsesActiveGoalsAndThreeDayCooldown(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	ctx := context.Background()
	q := store.Queries()

	workspace, err := q.GetOrCreateWorkspace(ctx, -100888000333, "Group", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	participant, err := q.RegisterParticipant(ctx, workspace.ID, 42, "igor", "Igor")
	if err != nil {
		t.Fatal(err)
	}
	period := domain.GoalPeriod{
		Key:      "summer-2026",
		Title:    "Лето 2026",
		StartsOn: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		EndsOn:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
	}
	if _, err := q.UpsertSeasonalGoalSet(ctx, workspace.ID, participant.ID, participant.UserID, period, "Результат: оффер\nМетрика: 10 откликов"); err != nil {
		t.Fatal(err)
	}

	firstSeed := nudgeSeed(t, participant.ID, period.Key, "first")
	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	text, err := appgoals.MaybeNudge(ctx, q, workspace, participant, firstSeed, string(domain.DailyTaskDone), now)
	if err != nil {
		t.Fatal(err)
	}
	if text == "" {
		t.Fatal("expected first nudge to be shown")
	}

	secondSeed := nudgeSeed(t, participant.ID, period.Key, "second")
	text, err = appgoals.MaybeNudge(ctx, q, workspace, participant, secondSeed, string(domain.DailyTaskFailed), now.Add(24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if text != "" {
		t.Fatalf("expected cooldown to suppress second nudge, got %q", text)
	}

	thirdSeed := nudgeSeed(t, participant.ID, period.Key, "third")
	text, err = appgoals.MaybeNudge(ctx, q, workspace, participant, thirdSeed, string(domain.DailyTaskFailed), now.Add(73*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if text == "" {
		t.Fatal("expected nudge after cooldown")
	}
}

func nudgeSeed(t *testing.T, participantID int64, periodKey string, prefix string) string {
	t.Helper()
	for i := range 10000 {
		seed := fmt.Sprintf("%s-%d", prefix, i)
		if domain.ShouldShowGoalNudge(fmt.Sprintf("%s:%d:%s", seed, participantID, periodKey)) {
			return seed
		}
	}
	t.Fatal("cannot find deterministic nudge seed")
	return ""
}
