package domain

import (
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

func TestParseRoutineItemsAcceptsSimpleBulletsAndNumbers(t *testing.T) {
	got, err := ParseRoutineItems("  - зарядка\n• работа\n1. английский перед сном\n2) йога\n\n")
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

func TestParseRoutineItemsRejectsTooManyItems(t *testing.T) {
	_, err := ParseRoutineItems("1\n2\n3\n4\n5\n6\n7\n8\n9\n10")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRoutineCheckinDueStartsNextMorning(t *testing.T) {
	created := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	_, due, err := RoutineCheckinDue(created, "UTC", time.Date(2026, 6, 23, 12, 1, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if due {
		t.Fatal("routine should not be due on creation day")
	}
	date, due, err := RoutineCheckinDue(created, "UTC", time.Date(2026, 6, 24, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if !due || date.Format("2006-01-02") != "2026-06-24" {
		t.Fatalf("due=%v date=%s", due, date.Format("2006-01-02"))
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

func TestGoalWeeklyReviewDueOnSundayEvening(t *testing.T) {
	_, due, err := GoalWeeklyReviewDue("UTC", time.Date(2026, 6, 28, 19, 59, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if due {
		t.Fatal("review should not be due before 20:00")
	}
	weekStart, due, err := GoalWeeklyReviewDue("UTC", time.Date(2026, 6, 28, 20, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if !due || weekStart.Format("2006-01-02") != "2026-06-22" {
		t.Fatalf("due=%v weekStart=%s", due, weekStart.Format("2006-01-02"))
	}
}

func TestGoalNudgeIsDeterministic(t *testing.T) {
	if ShouldShowGoalNudge("same-seed") != ShouldShowGoalNudge("same-seed") {
		t.Fatal("nudge decision must be stable for one seed")
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
