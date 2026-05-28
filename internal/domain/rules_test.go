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
