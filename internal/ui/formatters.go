package ui

import (
	"fmt"
	"html"
	"strings"

	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/storage/postgres"
)

const TodayControlText = "🎯 <b>Сегодня</b>\n" +
	"Здесь у каждого одна главная задача на день.\n" +
	"Нажми кнопку ниже, чтобы зафиксировать свой главный фокус.\n\n" +
	"Как это работает:\n" +
	"• ты формулируешь одну задачу на день;\n" +
	"• я закрепляю ее в отдельной карточке;\n" +
	"• вечером в этой же карточке можно оставить результат."

const ProgressIntroText = "✨ <b>Прогресс</b>\n" +
	"Здесь будет собираться все важное в аккуратную общую ленту.\n\n" +
	"Что появится здесь:\n" +
	"• закрытые задачи дня;\n" +
	"• автоматические итоги просроченных задач.\n\n" +
	"Так всегда видно, кто что сделал и довел до результата."

const SetupReadyText = "✅ <b>Все на месте.</b>\nТемы и стартовые сообщения уже в порядке. Ничего восстанавливать не пришлось."

const SetupRepairedText = "✨ <b>Готово!</b>\nЯ проверил пространство и восстановил все, чего не хватало.\n\n" +
	"Что дальше:\n" +
	"• в теме <b>Сегодня</b> каждый фиксирует одну задачу на день;\n" +
	"• в теме <b>Прогресс</b> будут появляться результаты."

func FormatSetupChecklist(ready bool, isSupergroup bool, isForum bool, isAdmin bool, canManageTopics bool, canReadMessages bool, notice string) string {
	status := "До запуска нужно закрыть несколько пунктов."
	if ready {
		status = "✅ Можно начинать: все условия выполнены."
	}
	lines := []string{
		"⚙️ <b>Подготовка пространства</b>",
		status,
		"",
		mark(isSupergroup) + " Группа переведена в супергруппу.",
		mark(isForum) + " Включены темы.",
		mark(isAdmin) + " Бот назначен администратором.",
		mark(canManageTopics) + " У бота есть право управлять темами.",
		mark(canReadMessages) + " Бот видит сообщения участников.",
		"",
		"Когда все будет готово, можно запускать оформление группы.",
	}
	return appendNotice(lines, notice)
}

func FormatDailyTaskCard(task postgres.DailyTask, displayName string, username string, notice string) string {
	person := personLabel(username, displayName)
	lines := []string{
		fmt.Sprintf("🎯 <b>Задача дня</b> %s:", person),
		"",
		renderSectionHTML(task.Text),
		"",
		"<b>Статус:</b> " + taskStatusLabel(task.Status),
	}
	if task.ReportText != nil && *task.ReportText != "" {
		lines = append(lines, "", "<b>Результат:</b>", renderSectionHTML(*task.ReportText))
	}
	return appendNotice(lines, notice)
}

func FormatProgressEvent(event postgres.ProgressEvent) string {
	payload := event.Payload
	person := profileLinkLabel(payload)
	switch event.EventType {
	case domain.ProgressDailyTaskClosed:
		task := payloadLink(payload, "task_link", "задачу дня")
		title := strings.Replace(dailyTaskClosedTitle(payloadString(payload, "status"), person), "задачу дня", task, 1)
		return strings.Join([]string{
			title,
			"",
			"<b>Что планировал:</b>",
			"",
			renderSectionHTML(payloadString(payload, "task_html")),
			"",
			"<b>Результат:</b>",
			"",
			renderSectionHTML(payloadString(payload, "report_html")),
		}, "\n")
	case domain.ProgressDailyTaskAutoFail:
		task := payloadLink(payload, "task_link", "задачу дня")
		return strings.Join([]string{
			fmt.Sprintf("⏰ <b>%s не выполнил %s вовремя</b>", person, task),
			"",
			"<b>Что планировал:</b>",
			"",
			renderSectionHTML(payloadString(payload, "task_html")),
		}, "\n")
	default:
		return "🔔 Системное сообщение\n" + fmt.Sprint(payload)
	}
}

func AlertText(kind domain.AlertKind) string {
	if kind == domain.AlertDayClosedPendingReport {
		return "🔔 День уже закончился, а отчет по задаче так и не появился."
	}
	return "⏰ Время вышло — задача автоматически отмечена как не выполненная."
}

func mark(value bool) string {
	if value {
		return "✅"
	}
	return "•"
}

func appendNotice(lines []string, notice string) string {
	if notice != "" {
		lines = append(lines, "", "<blockquote>"+html.EscapeString(notice)+"</blockquote>")
	}
	return strings.Join(lines, "\n")
}

func taskStatusLabel(status domain.DailyTaskStatus) string {
	switch status {
	case domain.DailyTaskActive:
		return "в процессе"
	case domain.DailyTaskAwaitingReport:
		return "ждет отчета"
	case domain.DailyTaskDone:
		return "выполнена"
	case domain.DailyTaskPartial:
		return "выполнена частично"
	case domain.DailyTaskFailed:
		return "не выполнена"
	default:
		return string(status)
	}
}

func dailyTaskClosedTitle(status string, person string) string {
	switch status {
	case "done":
		return fmt.Sprintf("✅ <b>%s выполнил задачу дня</b>", person)
	case "partial":
		return fmt.Sprintf("🔸 <b>%s частично выполнил задачу дня</b>", person)
	case "failed":
		return fmt.Sprintf("❌ <b>%s не выполнил задачу дня</b>", person)
	default:
		return fmt.Sprintf("✅ <b>%s завершил задачу дня</b>", person)
	}
}

func personLabel(username string, displayName string) string {
	if username != "" {
		return "@" + html.EscapeString(username)
	}
	return html.EscapeString(displayName)
}

func profileLinkLabel(payload map[string]any) string {
	displayName := payloadString(payload, "display_name")
	if displayName == "" {
		displayName = "Без имени"
	}
	label := html.EscapeString(displayName)
	if username := payloadString(payload, "username"); username != "" {
		label = html.EscapeString(username)
	}
	userID := payloadInt64(payload, "user_id")
	if userID == 0 {
		return label
	}
	return fmt.Sprintf(`<a href="tg://user?id=%d">%s</a>`, userID, label)
}

func payloadLink(payload map[string]any, key string, label string) string {
	link := payloadString(payload, key)
	if link == "" {
		return label
	}
	return fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(link), label)
}

func renderSectionHTML(value string) string {
	if value == "" {
		return "<i>—</i>"
	}
	if strings.Contains(value, "<blockquote") {
		return value
	}
	return "<blockquote>" + value + "</blockquote>"
}

func payloadString(payload map[string]any, key string) string {
	if value, ok := payload[key].(string); ok {
		return value
	}
	return ""
}

func payloadInt64(payload map[string]any, key string) int64 {
	switch value := payload[key].(type) {
	case float64:
		return int64(value)
	case int64:
		return value
	case int:
		return int64(value)
	default:
		return 0
	}
}
