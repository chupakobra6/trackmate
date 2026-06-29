package domain

import (
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	"github.com/igor/trackmate/internal/messages"
)

const (
	MaxRoutineItems         = 9
	RoutineCheckinHour      = 8
	RoutineReminderHour     = 20
	RoutineAutoFailHour     = 0
	RoutineNoticeMaxAge     = 24 * time.Hour
	GoalWeeklyReviewWeekday = time.Sunday
	GoalWeeklyReviewHour    = 20
	GoalReviewIntervalDays  = 14
	GoalNudgePercent        = 10
	GoalNudgeCooldown       = 72 * time.Hour
	PersonalAlertPercent    = 30
	PendingInputMaxAge      = 24 * time.Hour
)

type DailyTaskTransition struct {
	NewStatus                DailyTaskStatus
	ShouldEmitAutoFail       bool
	ShouldEmitAwaitingReport bool
}

type GoalPeriod struct {
	Key      string
	Title    string
	StartsOn time.Time
	EndsOn   time.Time
}

func ParseRoutineItems(raw string) ([]string, error) {
	lines := strings.Split(raw, "\n")
	items := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		var item string
		item, ok := parseRoutineItemLine(trimmed)
		if !ok {
			return nil, fmt.Errorf("routine list must use list prefixes")
		}
		if item == "" {
			return nil, fmt.Errorf("routine item is empty")
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("routine list is empty")
	}
	if len(items) > MaxRoutineItems {
		return nil, fmt.Errorf("routine list has %d items, max %d", len(items), MaxRoutineItems)
	}
	return items, nil
}

func parseRoutineItemLine(trimmed string) (string, bool) {
	switch {
	case strings.HasPrefix(trimmed, "-"):
		return strings.TrimSpace(strings.TrimPrefix(trimmed, "-")), true
	case strings.HasPrefix(trimmed, "—"):
		return strings.TrimSpace(strings.TrimPrefix(trimmed, "—")), true
	}
	for i, r := range trimmed {
		if r >= '0' && r <= '9' {
			continue
		}
		if i > 0 && (r == '.' || r == ')') {
			return strings.TrimSpace(trimmed[i+1:]), true
		}
		break
	}
	return "", false
}

func RoutineScore(status RoutineItemStatus) float64 {
	switch status {
	case RoutineItemDone:
		return 1
	case RoutineItemPartial:
		return 0.5
	default:
		return 0
	}
}

func RoutineCheckinDue(planCreatedAt time.Time, workspaceTimezone string, nowUTC time.Time) (time.Time, bool, error) {
	location, err := time.LoadLocation(workspaceTimezone)
	if err != nil {
		return time.Time{}, false, err
	}
	localNow := nowUTC.In(location)
	if localNow.Hour() < RoutineCheckinHour {
		return time.Time{}, false, nil
	}
	checkinDate, due := RoutinePreviousCheckinDate(planCreatedAt, workspaceTimezone, nowUTC)
	if !due {
		return time.Time{}, false, nil
	}
	return checkinDate, true, nil
}

func RoutinePreviousCheckinDate(planCreatedAt time.Time, workspaceTimezone string, nowUTC time.Time) (time.Time, bool) {
	location, err := time.LoadLocation(workspaceTimezone)
	if err != nil {
		return time.Time{}, false
	}
	localNow := nowUTC.In(location)
	year, month, day := localNow.Date()
	today := time.Date(year, month, day, 0, 0, 0, 0, location)
	checkinDate := today.AddDate(0, 0, -1)
	createdYear, createdMonth, createdDay := planCreatedAt.In(location).Date()
	createdDate := time.Date(createdYear, createdMonth, createdDay, 0, 0, 0, 0, location)
	if checkinDate.Before(createdDate) {
		return time.Time{}, false
	}
	return checkinDate, true
}

func RoutineReminderDue(checkinDate time.Time, workspaceTimezone string, reminderSentAt *time.Time, completedAt *time.Time, nowUTC time.Time) (bool, error) {
	if reminderSentAt != nil || completedAt != nil {
		return false, nil
	}
	location, err := time.LoadLocation(workspaceTimezone)
	if err != nil {
		return false, err
	}
	localNow := nowUTC.In(location)
	nextDay := routineCheckinLocalDate(checkinDate, location).AddDate(0, 0, 1)
	reminderAt := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), RoutineReminderHour, 0, 0, 0, location)
	deadlineDay := nextDay.AddDate(0, 0, 1)
	deadline := time.Date(deadlineDay.Year(), deadlineDay.Month(), deadlineDay.Day(), RoutineAutoFailHour, 0, 0, 0, location)
	return !localNow.Before(reminderAt) && localNow.Before(deadline), nil
}

func RoutineAutoFailDue(checkinDate time.Time, workspaceTimezone string, completedAt *time.Time, nowUTC time.Time) (bool, error) {
	if completedAt != nil {
		return false, nil
	}
	location, err := time.LoadLocation(workspaceTimezone)
	if err != nil {
		return false, err
	}
	localNow := nowUTC.In(location)
	nextDay := routineCheckinLocalDate(checkinDate, location).AddDate(0, 0, 2)
	deadline := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), RoutineAutoFailHour, 0, 0, 0, location)
	return !localNow.Before(deadline), nil
}

func routineCheckinLocalDate(checkinDate time.Time, location *time.Location) time.Time {
	year, month, day := checkinDate.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, location)
}

func CurrentGoalPeriod(workspaceTimezone string, nowUTC time.Time) (GoalPeriod, error) {
	location, err := time.LoadLocation(workspaceTimezone)
	if err != nil {
		return GoalPeriod{}, err
	}
	localNow := nowUTC.In(location)
	year, month, _ := localNow.Date()
	seasonYear := year
	isWinter := false
	var key, title string
	var startMonth, endMonth time.Month
	switch {
	case month >= time.March && month <= time.May:
		key = fmt.Sprintf("spring-%d", year)
		title = messages.Format("season.spring", "year", fmt.Sprint(year))
		startMonth, endMonth = time.March, time.June
	case month >= time.June && month <= time.August:
		key = fmt.Sprintf("summer-%d", year)
		title = messages.Format("season.summer", "year", fmt.Sprint(year))
		startMonth, endMonth = time.June, time.September
	case month >= time.September && month <= time.November:
		key = fmt.Sprintf("autumn-%d", year)
		title = messages.Format("season.autumn", "year", fmt.Sprint(year))
		startMonth, endMonth = time.September, time.December
	default:
		isWinter = true
		if month <= time.February {
			seasonYear = year - 1
		}
		key = fmt.Sprintf("winter-%d", seasonYear)
		title = messages.Format("season.winter", "start_year", fmt.Sprint(seasonYear), "end_year", fmt.Sprint(seasonYear+1))
		startMonth, endMonth = time.December, time.March
	}
	startYear := seasonYear
	endYear := seasonYear
	if isWinter {
		endYear = seasonYear + 1
	}
	startsOn := time.Date(startYear, startMonth, 1, 0, 0, 0, 0, location)
	endsOn := time.Date(endYear, endMonth, 1, 0, 0, 0, 0, location)
	return GoalPeriod{Key: key, Title: title, StartsOn: startsOn, EndsOn: endsOn}, nil
}

func GoalWeeklyReviewDue(periodStartsOn time.Time, workspaceTimezone string, nowUTC time.Time) (time.Time, bool, error) {
	location, err := time.LoadLocation(workspaceTimezone)
	if err != nil {
		return time.Time{}, false, err
	}
	localNow := nowUTC.In(location)
	if localNow.Weekday() != GoalWeeklyReviewWeekday || localNow.Hour() < GoalWeeklyReviewHour {
		return time.Time{}, false, nil
	}
	year, month, day := localNow.Date()
	localDate := time.Date(year, month, day, 0, 0, 0, 0, location)
	if !goalReviewDateDue(periodStartsOn, localDate, location) {
		return time.Time{}, false, nil
	}
	weekStart := localDate.AddDate(0, 0, -int((localNow.Weekday()+6)%7))
	return weekStart, true, nil
}

func GoalReviewCountdown(periodStartsOn time.Time, periodEndsOn time.Time, workspaceTimezone string, nowUTC time.Time) (int, int, error) {
	location, err := time.LoadLocation(workspaceTimezone)
	if err != nil {
		return 0, 0, err
	}
	localNow := nowUTC.In(location)
	year, month, day := localNow.Date()
	localDate := time.Date(year, month, day, 0, 0, 0, 0, location)
	endYear, endMonth, endDay := periodEndsOn.In(location).Date()
	endDate := time.Date(endYear, endMonth, endDay, 0, 0, 0, 0, location)
	if !localDate.Before(endDate) {
		return 0, 0, nil
	}
	daysLeft := int(endDate.Sub(localDate).Hours() / 24)
	reviewsLeft := 0
	for date := localDate.AddDate(0, 0, 1); date.Before(endDate); date = date.AddDate(0, 0, 1) {
		if date.Weekday() == GoalWeeklyReviewWeekday && goalReviewDateDue(periodStartsOn, date, location) {
			reviewsLeft++
		}
	}
	return daysLeft, reviewsLeft, nil
}

func goalReviewDateDue(periodStartsOn time.Time, localDate time.Time, location *time.Location) bool {
	startYear, startMonth, startDay := periodStartsOn.In(location).Date()
	startDate := time.Date(startYear, startMonth, startDay, 0, 0, 0, 0, location)
	daysSinceStart := int(localDate.Sub(startDate).Hours() / 24)
	if daysSinceStart < GoalReviewIntervalDays-1 {
		return false
	}
	reviewWeekIndex := (daysSinceStart - (GoalReviewIntervalDays - 1)) / 7
	return reviewWeekIndex%2 == 0
}

func GoalFinalReviewDue(period GoalPeriod, workspaceTimezone string, nowUTC time.Time) (bool, error) {
	location, err := time.LoadLocation(workspaceTimezone)
	if err != nil {
		return false, err
	}
	localNow := nowUTC.In(location)
	year, month, day := localNow.Date()
	localDate := time.Date(year, month, day, 0, 0, 0, 0, location)
	endYear, endMonth, endDay := period.EndsOn.In(location).Date()
	endDate := time.Date(endYear, endMonth, endDay, 0, 0, 0, 0, location)
	return !localDate.Before(endDate), nil
}

func ShouldShowGoalNudge(seed string) bool {
	return shouldShowPercent(seed, GoalNudgePercent)
}

func shouldShowPercent(seed string, percent int) bool {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(seed))
	return hash.Sum32()%100 < uint32(percent)
}

func ShouldShowPersonalAlert(username string, seed string) bool {
	if !isPersonalAlertTarget(username) {
		return false
	}
	return shouldShowPercent(normalizeUsername(username)+":"+seed, PersonalAlertPercent)
}

func isPersonalAlertTarget(username string) bool {
	return normalizeUsername(username) == "whysoxxx"
}

func normalizeUsername(username string) string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(username)), "@")
}

func GoalNudgeAllowed(lastShownAt *time.Time, nowUTC time.Time) bool {
	if lastShownAt == nil {
		return true
	}
	return !nowUTC.Before(lastShownAt.UTC().Add(GoalNudgeCooldown))
}

func NextDailyTaskTransition(taskDate time.Time, workspaceTimezone string, currentStatus DailyTaskStatus, nowUTC time.Time) (DailyTaskTransition, error) {
	location, err := time.LoadLocation(workspaceTimezone)
	if err != nil {
		return DailyTaskTransition{}, err
	}
	localNow := nowUTC.In(location)
	year, month, day := taskDate.In(location).Date()
	nextDay := time.Date(year, month, day+1, 0, 0, 0, 0, location)
	noon := time.Date(year, month, day+1, 12, 0, 0, 0, location)

	if currentStatus == DailyTaskActive && !localNow.Before(noon) {
		return DailyTaskTransition{NewStatus: DailyTaskFailed, ShouldEmitAutoFail: true}, nil
	}
	if currentStatus == DailyTaskActive && !localNow.Before(nextDay) {
		return DailyTaskTransition{NewStatus: DailyTaskAwaitingReport, ShouldEmitAwaitingReport: true}, nil
	}
	if currentStatus == DailyTaskAwaitingReport && !localNow.Before(noon) {
		return DailyTaskTransition{NewStatus: DailyTaskFailed, ShouldEmitAutoFail: true}, nil
	}
	return DailyTaskTransition{}, nil
}

func LocalTaskDate(timezoneName string, nowUTC time.Time) (time.Time, error) {
	location, err := time.LoadLocation(timezoneName)
	if err != nil {
		return time.Time{}, err
	}
	local := nowUTC.In(location)
	year, month, day := local.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, location), nil
}
