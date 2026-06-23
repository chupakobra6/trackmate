package domain

import (
	"fmt"
	"hash/fnv"
	"regexp"
	"strings"
	"time"
)

const (
	MaxRoutineItems         = 9
	RoutineCheckinHour      = 9
	GoalWeeklyReviewWeekday = time.Sunday
	GoalWeeklyReviewHour    = 20
	GoalNudgePercent        = 10
	GoalNudgeCooldown       = 72 * time.Hour
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

var routineLinePrefix = regexp.MustCompile(`^\s*(?:[-*•]\s*|\d+[\.)]\s*)`)

func ParseRoutineItems(raw string) ([]string, error) {
	lines := strings.Split(raw, "\n")
	items := make([]string, 0, len(lines))
	for _, line := range lines {
		item := strings.TrimSpace(routineLinePrefix.ReplaceAllString(line, ""))
		if item == "" {
			continue
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
	year, month, day := localNow.Date()
	checkinDate := time.Date(year, month, day, 0, 0, 0, 0, location)
	createdYear, createdMonth, createdDay := planCreatedAt.In(location).Date()
	createdDate := time.Date(createdYear, createdMonth, createdDay, 0, 0, 0, 0, location)
	return checkinDate, checkinDate.After(createdDate), nil
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
		key, title = fmt.Sprintf("spring-%d", year), fmt.Sprintf("Весна %d", year)
		startMonth, endMonth = time.March, time.June
	case month >= time.June && month <= time.August:
		key, title = fmt.Sprintf("summer-%d", year), fmt.Sprintf("Лето %d", year)
		startMonth, endMonth = time.June, time.September
	case month >= time.September && month <= time.November:
		key, title = fmt.Sprintf("autumn-%d", year), fmt.Sprintf("Осень %d", year)
		startMonth, endMonth = time.September, time.December
	default:
		isWinter = true
		if month <= time.February {
			seasonYear = year - 1
		}
		key, title = fmt.Sprintf("winter-%d", seasonYear), fmt.Sprintf("Зима %d/%d", seasonYear, seasonYear+1)
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

func GoalWeeklyReviewDue(workspaceTimezone string, nowUTC time.Time) (time.Time, bool, error) {
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
	weekStart := localDate.AddDate(0, 0, -int((localNow.Weekday()+6)%7))
	return weekStart, true, nil
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
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(seed))
	return hash.Sum32()%100 < GoalNudgePercent
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
