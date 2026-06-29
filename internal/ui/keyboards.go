package ui

import (
	"fmt"

	"github.com/igor/trackmate/internal/messages"
	"github.com/igor/trackmate/internal/telegram"
)

func SetupKeyboard() *telegram.InlineKeyboardMarkup {
	return &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{
		{{Text: messages.Text("button.setup.check"), CallbackData: "setup:check"}},
		{{Text: messages.Text("button.setup.start"), CallbackData: "setup:start"}},
	}}
}

func TodayControlKeyboard() *telegram.InlineKeyboardMarkup {
	return &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{
		{{Text: messages.Text("button.today.add"), CallbackData: "today:add"}},
	}}
}

func RoutineControlKeyboard() *telegram.InlineKeyboardMarkup {
	return &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{
		{{Text: messages.Text("button.routine.configure"), CallbackData: "routine:configure"}},
	}}
}

func GoalsControlKeyboard() *telegram.InlineKeyboardMarkup {
	return &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{
		{{Text: messages.Text("button.goals.configure"), CallbackData: "goals:configure"}},
	}}
}

func EmptyKeyboard() *telegram.InlineKeyboardMarkup {
	return &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{}}
}

func RoutineItemKeyboard(checkinID int64, itemIndex int) *telegram.InlineKeyboardMarkup {
	return &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{
		{
			{Text: messages.Text("button.routine.done"), CallbackData: fmt.Sprintf("routine:item:%d:%d:done", checkinID, itemIndex)},
			{Text: messages.Text("button.routine.partial"), CallbackData: fmt.Sprintf("routine:item:%d:%d:partial", checkinID, itemIndex)},
			{Text: messages.Text("button.routine.failed"), CallbackData: fmt.Sprintf("routine:item:%d:%d:failed", checkinID, itemIndex)},
		},
	}}
}

func GoalFinalStatusKeyboard(goalSetID int64) *telegram.InlineKeyboardMarkup {
	return &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{
		{
			{Text: messages.Text("button.goal.done"), CallbackData: fmt.Sprintf("goals:final:%d:done", goalSetID)},
			{Text: messages.Text("button.goal.partial"), CallbackData: fmt.Sprintf("goals:final:%d:partial", goalSetID)},
			{Text: messages.Text("button.goal.failed"), CallbackData: fmt.Sprintf("goals:final:%d:failed", goalSetID)},
		},
	}}
}

func DailyTaskKeyboard(taskID int64) *telegram.InlineKeyboardMarkup {
	return &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{
		{{Text: messages.Text("button.task.report"), CallbackData: fmt.Sprintf("task:report:%d", taskID)}},
	}}
}

func DailyTaskStatusKeyboard(taskID int64) *telegram.InlineKeyboardMarkup {
	return &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{
		{
			{Text: messages.Text("button.task.done"), CallbackData: fmt.Sprintf("task:status:%d:done", taskID)},
			{Text: messages.Text("button.task.partial"), CallbackData: fmt.Sprintf("task:status:%d:partial", taskID)},
			{Text: messages.Text("button.task.failed"), CallbackData: fmt.Sprintf("task:status:%d:failed", taskID)},
		},
	}}
}

func AlertKeyboard(taskID int64, alertID int64) *telegram.InlineKeyboardMarkup {
	return &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{
		{{Text: messages.Text("button.task.report"), CallbackData: fmt.Sprintf("task:report:%d", taskID)}},
		{{Text: messages.Text("button.dismiss"), CallbackData: fmt.Sprintf("alert:ack:%d", alertID)}},
	}}
}

func DismissKeyboard() *telegram.InlineKeyboardMarkup {
	return &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{
		{{Text: messages.Text("button.dismiss"), CallbackData: "notice:dismiss"}},
	}}
}
