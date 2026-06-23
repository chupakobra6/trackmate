package ui

import (
	"fmt"

	"github.com/igor/trackmate/internal/telegram"
)

func SetupKeyboard() *telegram.InlineKeyboardMarkup {
	return &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{
		{{Text: "🔄 Проверить снова", CallbackData: "setup:check"}},
		{{Text: "✨ Оформить группу", CallbackData: "setup:start"}},
	}}
}

func TodayControlKeyboard() *telegram.InlineKeyboardMarkup {
	return &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{
		{{Text: "➕ Добавить задачу", CallbackData: "today:add"}},
	}}
}

func RoutineControlKeyboard() *telegram.InlineKeyboardMarkup {
	return &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{
		{{Text: "✏️ Настроить рутину", CallbackData: "routine:configure"}},
	}}
}

func GoalsControlKeyboard() *telegram.InlineKeyboardMarkup {
	return &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{
		{{Text: "✏️ Настроить цели", CallbackData: "goals:configure"}},
	}}
}

func RoutineItemKeyboard(checkinID int64, itemIndex int) *telegram.InlineKeyboardMarkup {
	return &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{
		{
			{Text: "✅ Да", CallbackData: fmt.Sprintf("routine:item:%d:%d:done", checkinID, itemIndex)},
			{Text: "🔸 Частично", CallbackData: fmt.Sprintf("routine:item:%d:%d:partial", checkinID, itemIndex)},
			{Text: "❌ Нет", CallbackData: fmt.Sprintf("routine:item:%d:%d:failed", checkinID, itemIndex)},
		},
	}}
}

func GoalFinalStatusKeyboard(goalSetID int64) *telegram.InlineKeyboardMarkup {
	return &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{
		{
			{Text: "✅ Выполнены", CallbackData: fmt.Sprintf("goals:final:%d:done", goalSetID)},
			{Text: "🔸 Частично", CallbackData: fmt.Sprintf("goals:final:%d:partial", goalSetID)},
			{Text: "❌ Не выполнены", CallbackData: fmt.Sprintf("goals:final:%d:failed", goalSetID)},
		},
	}}
}

func DailyTaskKeyboard(taskID int64) *telegram.InlineKeyboardMarkup {
	return &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{
		{{Text: "🏁 Подвести итог", CallbackData: fmt.Sprintf("task:report:%d", taskID)}},
	}}
}

func DailyTaskStatusKeyboard(taskID int64) *telegram.InlineKeyboardMarkup {
	return &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{
		{
			{Text: "✅ Выполнена", CallbackData: fmt.Sprintf("task:status:%d:done", taskID)},
			{Text: "🔸 Выполнена частично", CallbackData: fmt.Sprintf("task:status:%d:partial", taskID)},
			{Text: "❌ Не выполнена", CallbackData: fmt.Sprintf("task:status:%d:failed", taskID)},
		},
	}}
}

func AlertKeyboard(taskID int64, alertID int64) *telegram.InlineKeyboardMarkup {
	return &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{
		{{Text: "🏁 Подвести итог", CallbackData: fmt.Sprintf("task:report:%d", taskID)}},
		{{Text: "👀 Понял", CallbackData: fmt.Sprintf("alert:ack:%d", alertID)}},
	}}
}
