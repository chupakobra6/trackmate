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

func DailyTaskKeyboard(taskID int64) *telegram.InlineKeyboardMarkup {
	return &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{
		{{Text: "🏁 Отчитаться", CallbackData: fmt.Sprintf("task:report:%d", taskID)}},
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
		{{Text: "🏁 Отчитаться", CallbackData: fmt.Sprintf("task:report:%d", taskID)}},
		{{Text: "👀 Понял", CallbackData: fmt.Sprintf("alert:ack:%d", alertID)}},
	}}
}
