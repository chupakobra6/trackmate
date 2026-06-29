package domain

import (
	"fmt"
	"testing"
	"time"
)

func TestLocalTaskDateUsesWorkspaceTimezone(t *testing.T) {
	now := time.Date(2026, 4, 7, 21, 30, 0, 0, time.UTC)
	got, err := LocalTaskDate("Europe/Moscow", now)
	if err != nil {
		t.Fatal(err)
	}
	if got.Format("2006-01-02") != "2026-04-08" {
		t.Fatalf("unexpected date %s", got.Format("2006-01-02"))
	}
}

func TestDailyTaskTransitions(t *testing.T) {
	taskDate := time.Date(2026, 4, 7, 0, 0, 0, 0, time.UTC)
	awaiting, err := NextDailyTaskTransition(taskDate, "UTC", DailyTaskActive, time.Date(2026, 4, 8, 0, 0, 1, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if awaiting.NewStatus != DailyTaskAwaitingReport || !awaiting.ShouldEmitAwaitingReport {
		t.Fatalf("unexpected awaiting transition: %+v", awaiting)
	}
	failed, err := NextDailyTaskTransition(taskDate, "UTC", DailyTaskAwaitingReport, time.Date(2026, 4, 8, 12, 0, 1, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if failed.NewStatus != DailyTaskFailed || !failed.ShouldEmitAutoFail {
		t.Fatalf("unexpected failed transition: %+v", failed)
	}
}

func TestParseRoutineItemsAcceptsDashLines(t *testing.T) {
	got, err := ParseRoutineItems("  - зарядка\n— работа\n- английский перед сном\n- йога\n\n")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"зарядка", "работа", "английский перед сном", "йога"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestParseRoutineItemsAcceptsNumberedLines(t *testing.T) {
	got, err := ParseRoutineItems("1. зарядка\n12. работа\n3) английский перед сном")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"зарядка", "работа", "английский перед сном"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestParseRoutineItemsRejectsPlainLinesAndUnsupportedBullets(t *testing.T) {
	for _, input := range []string{
		"зарядка\n- работа",
		"• зарядка\n- работа",
	} {
		if _, err := ParseRoutineItems(input); err == nil {
			t.Fatalf("expected error for %q", input)
		}
	}
}

func TestParseRoutineItemsRejectsTooManyItems(t *testing.T) {
	_, err := ParseRoutineItems("- 1\n- 2\n- 3\n- 4\n- 5\n- 6\n- 7\n- 8\n- 9\n- 10")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRoutineCheckinDueSendsNextMorningForPreviousDay(t *testing.T) {
	created := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	_, due, err := RoutineCheckinDue(created, "UTC", time.Date(2026, 6, 24, 7, 59, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if due {
		t.Fatal("routine should not be due before morning dispatch")
	}
	date, due, err := RoutineCheckinDue(created, "UTC", time.Date(2026, 6, 24, 8, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if !due || date.Format("2006-01-02") != "2026-06-23" {
		t.Fatalf("due=%v date=%s", due, date.Format("2006-01-02"))
	}
}

func TestRoutineCheckinDueSkipsDaysBeforePlanExists(t *testing.T) {
	created := time.Date(2026, 6, 24, 7, 0, 0, 0, time.UTC)
	_, due, err := RoutineCheckinDue(created, "UTC", time.Date(2026, 6, 24, 8, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if due {
		t.Fatal("routine should not create a check-in for a day before the plan existed")
	}
	date, due, err := RoutineCheckinDue(created, "UTC", time.Date(2026, 6, 25, 8, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if !due || date.Format("2006-01-02") != "2026-06-24" {
		t.Fatalf("due=%v date=%s", due, date.Format("2006-01-02"))
	}
}

func TestRoutineReminderAndAutoFailDue(t *testing.T) {
	checkinDate := time.Date(2026, 6, 23, 0, 0, 0, 0, time.UTC)
	reminder, err := RoutineReminderDue(checkinDate, "UTC", nil, nil, time.Date(2026, 6, 24, 19, 59, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if reminder {
		t.Fatal("routine should not remind before 20:00")
	}
	reminder, err = RoutineReminderDue(checkinDate, "UTC", nil, nil, time.Date(2026, 6, 24, 20, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if !reminder {
		t.Fatal("expected reminder at 20:00 next day")
	}
	autoFail, err := RoutineAutoFailDue(checkinDate, "UTC", nil, time.Date(2026, 6, 24, 23, 59, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if autoFail {
		t.Fatal("routine should not auto-close before midnight")
	}
	autoFail, err = RoutineAutoFailDue(checkinDate, "UTC", nil, time.Date(2026, 6, 25, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if !autoFail {
		t.Fatal("expected routine auto-close at midnight after the check-in day")
	}
}

func TestCurrentGoalPeriodReturnsSummer2026(t *testing.T) {
	period, err := CurrentGoalPeriod("Europe/Moscow", time.Date(2026, 6, 23, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if period.Key != "summer-2026" || period.Title != "Лето 2026" || period.EndsOn.Format("2006-01-02") != "2026-09-01" {
		t.Fatalf("unexpected period: %+v", period)
	}
}

func TestGoalWeeklyReviewDueEveryOtherSundayEvening(t *testing.T) {
	periodStart := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	_, due, err := GoalWeeklyReviewDue(periodStart, "UTC", time.Date(2026, 6, 28, 19, 59, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if due {
		t.Fatal("review should not be due before 20:00")
	}
	_, due, err = GoalWeeklyReviewDue(periodStart, "UTC", time.Date(2026, 6, 21, 20, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if due {
		t.Fatal("review should skip the off week")
	}
	weekStart, due, err := GoalWeeklyReviewDue(periodStart, "UTC", time.Date(2026, 6, 28, 20, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if !due || weekStart.Format("2006-01-02") != "2026-06-22" {
		t.Fatalf("due=%v weekStart=%s", due, weekStart.Format("2006-01-02"))
	}
}

func TestGoalReviewCountdownCountsFutureReviewsAndDays(t *testing.T) {
	days, reviews, err := GoalReviewCountdown(
		time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		"UTC",
		time.Date(2026, 6, 28, 20, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	if days != 65 || reviews != 4 {
		t.Fatalf("days=%d reviews=%d", days, reviews)
	}
}

func TestGoalNudgeIsDeterministic(t *testing.T) {
	if ShouldShowGoalNudge("same-seed") != ShouldShowGoalNudge("same-seed") {
		t.Fatal("nudge decision must be stable for one seed")
	}
}

func TestPersonalAlertTargetsOnlyExactEgorUsername(t *testing.T) {
	if !isPersonalAlertTarget("whysoxxx") {
		t.Fatal("expected exact Egor username to be a personal alert target")
	}
	if !isPersonalAlertTarget("@whysoxxx") {
		t.Fatal("expected username normalization to accept @ prefix")
	}
	for _, username := range []string{"w", "whysoxxx1", "igor", ""} {
		if isPersonalAlertTarget(username) {
			t.Fatalf("unexpected personal alert target: %s", username)
		}
	}
}

func TestPersonalAlertUsesStableThirtyPercentBucket(t *testing.T) {
	if ShouldShowPersonalAlert("whysoxxx", "same-seed") != ShouldShowPersonalAlert("whysoxxx", "same-seed") {
		t.Fatal("personal alert decision must be stable for one seed")
	}
	shown := 0
	for i := 0; i < 1000; i++ {
		if ShouldShowPersonalAlert("whysoxxx", fmt.Sprintf("seed-%d", i)) {
			shown++
		}
	}
	if shown < 250 || shown > 350 {
		t.Fatalf("unexpected personal alert share: %d/1000", shown)
	}
	if ShouldShowPersonalAlert("whysoxxx1", "seed-1") {
		t.Fatal("personal alert should not show for non-target username")
	}
}

func TestGoalNudgeAllowedUsesThreeDayCooldown(t *testing.T) {
	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	if !GoalNudgeAllowed(nil, now) {
		t.Fatal("nudge should be allowed without previous show")
	}
	last := now.Add(-71 * time.Hour)
	if GoalNudgeAllowed(&last, now) {
		t.Fatal("nudge should be blocked inside cooldown")
	}
	last = now.Add(-72 * time.Hour)
	if !GoalNudgeAllowed(&last, now) {
		t.Fatal("nudge should be allowed after cooldown")
	}
}

func TestGoalFinalReviewDueUsesWorkspaceLocalDate(t *testing.T) {
	period := GoalPeriod{EndsOn: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)}
	due, err := GoalFinalReviewDue(period, "Europe/Moscow", time.Date(2026, 8, 31, 21, 5, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if !due {
		t.Fatal("expected final review after local midnight on period end date")
	}
}
