package domain

import (
	"fmt"
	"strconv"
	"strings"
)

type CallbackKind string

const (
	CallbackUnknown          CallbackKind = "unknown"
	CallbackSetupCheck       CallbackKind = "setup_check"
	CallbackSetupStart       CallbackKind = "setup_start"
	CallbackTodayAdd         CallbackKind = "today_add"
	CallbackTaskReport       CallbackKind = "task_report"
	CallbackTaskStatus       CallbackKind = "task_status"
	CallbackAlertAck         CallbackKind = "alert_ack"
	CallbackRoutineConfigure CallbackKind = "routine_configure"
	CallbackRoutineItem      CallbackKind = "routine_item"
	CallbackGoalsConfigure   CallbackKind = "goals_configure"
	CallbackGoalFinalStatus  CallbackKind = "goal_final_status"
)

type Callback struct {
	Kind              CallbackKind
	TaskID            int64
	AlertID           int64
	TaskStatus        DailyTaskStatus
	RoutineCheckinID  int64
	RoutineItemIndex  int
	RoutineItemStatus RoutineItemStatus
	GoalSetID         int64
	GoalFinalStatus   GoalFinalStatus
	Raw               string
}

func ParseCallback(raw string) (Callback, error) {
	if len(raw) > 64 {
		return Callback{Kind: CallbackUnknown, Raw: raw}, fmt.Errorf("callback too long")
	}
	switch raw {
	case "setup:check":
		return Callback{Kind: CallbackSetupCheck, Raw: raw}, nil
	case "setup:start":
		return Callback{Kind: CallbackSetupStart, Raw: raw}, nil
	case "today:add":
		return Callback{Kind: CallbackTodayAdd, Raw: raw}, nil
	case "routine:configure":
		return Callback{Kind: CallbackRoutineConfigure, Raw: raw}, nil
	case "goals:configure":
		return Callback{Kind: CallbackGoalsConfigure, Raw: raw}, nil
	}

	parts := strings.Split(raw, ":")
	switch {
	case len(parts) == 3 && parts[0] == "task" && parts[1] == "report":
		id, err := parsePositiveID(parts[2])
		if err != nil {
			return Callback{Kind: CallbackUnknown, Raw: raw}, err
		}
		return Callback{Kind: CallbackTaskReport, TaskID: id, Raw: raw}, nil
	case len(parts) == 4 && parts[0] == "task" && parts[1] == "status":
		id, err := parsePositiveID(parts[2])
		if err != nil {
			return Callback{Kind: CallbackUnknown, Raw: raw}, err
		}
		status := DailyTaskStatus(parts[3])
		if !status.IsFinalReport() {
			return Callback{Kind: CallbackUnknown, Raw: raw}, fmt.Errorf("unknown task status %q", parts[3])
		}
		return Callback{Kind: CallbackTaskStatus, TaskID: id, TaskStatus: status, Raw: raw}, nil
	case len(parts) == 3 && parts[0] == "alert" && parts[1] == "ack":
		id, err := parsePositiveID(parts[2])
		if err != nil {
			return Callback{Kind: CallbackUnknown, Raw: raw}, err
		}
		return Callback{Kind: CallbackAlertAck, AlertID: id, Raw: raw}, nil
	case len(parts) == 5 && parts[0] == "routine" && parts[1] == "item":
		id, err := parsePositiveID(parts[2])
		if err != nil {
			return Callback{Kind: CallbackUnknown, Raw: raw}, err
		}
		index, err := strconv.Atoi(parts[3])
		if err != nil || index < 0 {
			if err == nil {
				err = fmt.Errorf("index must be non-negative")
			}
			return Callback{Kind: CallbackUnknown, Raw: raw}, err
		}
		status := RoutineItemStatus(parts[4])
		if !status.IsValid() {
			return Callback{Kind: CallbackUnknown, Raw: raw}, fmt.Errorf("unknown routine status %q", parts[4])
		}
		return Callback{Kind: CallbackRoutineItem, RoutineCheckinID: id, RoutineItemIndex: index, RoutineItemStatus: status, Raw: raw}, nil
	case len(parts) == 4 && parts[0] == "goals" && parts[1] == "final":
		id, err := parsePositiveID(parts[2])
		if err != nil {
			return Callback{Kind: CallbackUnknown, Raw: raw}, err
		}
		status := GoalFinalStatus(parts[3])
		if !status.IsValid() {
			return Callback{Kind: CallbackUnknown, Raw: raw}, fmt.Errorf("unknown goal final status %q", parts[3])
		}
		return Callback{Kind: CallbackGoalFinalStatus, GoalSetID: id, GoalFinalStatus: status, Raw: raw}, nil
	default:
		return Callback{Kind: CallbackUnknown, Raw: raw}, fmt.Errorf("unknown callback")
	}
}

func parsePositiveID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		if err == nil {
			err = fmt.Errorf("id must be positive")
		}
		return 0, err
	}
	return id, nil
}
