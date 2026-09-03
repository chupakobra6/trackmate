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

func TestIsDailySummaryTimeUsesWorkspaceTimezoneAndCutoff(t *testing.T) {
	before, err := IsDailySummaryTime("Europe/Moscow", time.Date(2026, 7, 12, 16, 59, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if before {
		t.Fatal("summary mode should stay off before 20:00 local time")
	}

	atCutoff, err := IsDailySummaryTime("Europe/Moscow", time.Date(2026, 7, 12, 17, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if !atCutoff {
		t.Fatal("summary mode should start exactly at 20:00 local time")
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

func TestRoutinePlanChangeSnapshotDateUsesActiveRoutineDay(t *testing.T) {
	created := time.Date(2026, 6, 30, 21, 35, 0, 0, time.UTC) // 1 July, 00:35 MSK.
	date, ok := RoutinePlanChangeSnapshotDate(created, "Europe/Moscow", time.Date(2026, 7, 1, 13, 3, 0, 0, time.UTC))
	if !ok || date.Format("2006-01-02") != "2026-07-01" {
		t.Fatalf("daytime routine edit should snapshot active local day, ok=%v date=%s", ok, date.Format("2006-01-02"))
	}

	date, ok = RoutinePlanChangeSnapshotDate(created, "Europe/Moscow", time.Date(2026, 7, 2, 4, 59, 0, 0, time.UTC))
	if !ok || date.Format("2006-01-02") != "2026-07-01" {
		t.Fatalf("early morning routine edit should snapshot previous local day, ok=%v date=%s", ok, date.Format("2006-01-02"))
	}

	_, ok = RoutinePlanChangeSnapshotDate(time.Date(2026, 7, 1, 22, 0, 0, 0, time.UTC), "Europe/Moscow", time.Date(2026, 7, 1, 23, 0, 0, 0, time.UTC))
	if ok {
		t.Fatal("routine edit should not snapshot a day before the plan existed")
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

func TestCurrentGoalPeriodChangesAtEveryLocalSeasonBoundary(t *testing.T) {
	tests := []struct {
		name string
		now  time.Time
		key  string
	}{
		{name: "spring", now: time.Date(2026, 3, 1, 8, 0, 0, 0, time.UTC), key: "spring-2026"},
		{name: "summer", now: time.Date(2026, 6, 1, 7, 0, 0, 0, time.UTC), key: "summer-2026"},
		{name: "autumn", now: time.Date(2026, 9, 1, 7, 0, 0, 0, time.UTC), key: "autumn-2026"},
		{name: "winter", now: time.Date(2026, 12, 1, 8, 0, 0, 0, time.UTC), key: "winter-2026"},
		{name: "next spring", now: time.Date(2027, 3, 1, 8, 0, 0, 0, time.UTC), key: "spring-2027"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			period, err := CurrentGoalPeriod("America/Los_Angeles", test.now)
			if err != nil {
				t.Fatal(err)
			}
			if period.Key != test.key {
				t.Fatalf("period=%s want=%s", period.Key, test.key)
			}
		})
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

func TestGoalWeeklyReviewLifecycleActionBoundaries(t *testing.T) {
	if retryAge := GoalReviewSkipAfter - GoalReviewReminderDelay; retryAge >= 48*time.Hour {
		t.Fatalf("retry age=%s must stay inside Telegram's 48-hour delete window", retryAge)
	}
	requestedAt := time.Date(2026, 8, 9, 17, 0, 0, 0, time.UTC)
	remindedAt := requestedAt.Add(GoalReviewReminderDelay)
	respondedAt := requestedAt.Add(time.Hour)
	skippedAt := requestedAt.Add(GoalReviewSkipAfter)

	tests := []struct {
		name       string
		now        time.Time
		remindedAt *time.Time
		responded  *time.Time
		skipped    *time.Time
		want       GoalWeeklyReviewAction
	}{
		{name: "before reminder", now: requestedAt.Add(GoalReviewReminderDelay - time.Second), want: GoalWeeklyReviewNoAction},
		{name: "at reminder", now: requestedAt.Add(GoalReviewReminderDelay), want: GoalWeeklyReviewRemind},
		{name: "after one reminder", now: requestedAt.Add(48 * time.Hour), remindedAt: &remindedAt, want: GoalWeeklyReviewNoAction},
		{name: "at skip without reminder", now: requestedAt.Add(GoalReviewSkipAfter), want: GoalWeeklyReviewSkip},
		{name: "at skip after reminder", now: requestedAt.Add(GoalReviewSkipAfter), remindedAt: &remindedAt, want: GoalWeeklyReviewSkip},
		{name: "already responded", now: requestedAt.Add(GoalReviewSkipAfter), responded: &respondedAt, want: GoalWeeklyReviewNoAction},
		{name: "already skipped", now: requestedAt.Add(GoalReviewSkipAfter + time.Hour), skipped: &skippedAt, want: GoalWeeklyReviewNoAction},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GoalWeeklyReviewLifecycleAction(requestedAt, tt.remindedAt, tt.responded, tt.skipped, tt.now); got != tt.want {
				t.Fatalf("action=%q want=%q", got, tt.want)
			}
		})
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

func TestGoalDatesKeepDatabaseCalendarDayInNegativeTimezone(t *testing.T) {
	period := GoalPeriod{
		StartsOn: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		EndsOn:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
	}
	before, err := GoalFinalReviewDue(period, "America/Los_Angeles", time.Date(2026, 9, 1, 6, 59, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if before {
		t.Fatal("final review became due before September 1 local time")
	}
	due, err := GoalFinalReviewDue(period, "America/Los_Angeles", time.Date(2026, 9, 1, 7, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if !due {
		t.Fatal("final review was not due on the stored September 1 calendar date")
	}
	days, _, err := GoalReviewCountdown(period.StartsOn, period.EndsOn, "America/Los_Angeles", time.Date(2026, 8, 31, 19, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if days != 1 {
		t.Fatalf("days=%d want=1", days)
	}
}
