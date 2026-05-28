package domain

import (
	"time"
)

type DailyTaskTransition struct {
	NewStatus                DailyTaskStatus
	ShouldEmitAutoFail       bool
	ShouldEmitAwaitingReport bool
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
