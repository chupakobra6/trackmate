package domain

import (
	"fmt"
	"strconv"
	"strings"
)

type CallbackKind string

const (
	CallbackUnknown    CallbackKind = "unknown"
	CallbackSetupCheck CallbackKind = "setup_check"
	CallbackSetupStart CallbackKind = "setup_start"
	CallbackTodayAdd   CallbackKind = "today_add"
	CallbackTaskReport CallbackKind = "task_report"
	CallbackTaskStatus CallbackKind = "task_status"
	CallbackAlertAck   CallbackKind = "alert_ack"
)

type Callback struct {
	Kind       CallbackKind
	TaskID     int64
	AlertID    int64
	TaskStatus DailyTaskStatus
	Raw        string
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
