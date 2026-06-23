package ui

import (
	"fmt"
	"html"
	"strings"

	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/storage/postgres"
)

const TodayControlText = "🎯 <b>Сегодня</b>\n" +
	"Здесь у каждого одна главная цель-задача дня.\n" +
	"Нажми кнопку ниже, чтобы зафиксировать свой главный фокус.\n\n" +
	"Как это работает:\n" +
	"• ты формулируешь одну главную цель-задачу дня;\n" +
	"• я закрепляю ее в отдельной карточке;\n" +
	"• вечером в этой же карточке можно оставить результат."

const ProgressIntroText = "✨ <b>Прогресс</b>\n" +
	"Здесь будет собираться все важное в аккуратную общую ленту.\n\n" +
	"Что появится здесь:\n" +
	"• закрытые задачи дня;\n" +
	"• автоматические итоги просроченных задач.\n\n" +
	"Так всегда видно, кто что сделал и довел до результата."

const RoutineControlText = "🔁 <b>Рутины</b>\n" +
	"Здесь живут повторяющиеся действия: зарядка, английский, йога, режим и другие ежедневные опоры.\n\n" +
	"Нажми кнопку ниже и пришли список. Я буду спрашивать по нему один раз в день одной карточкой."

const GoalsControlText = "🎯 <b>Цели</b>\n" +
	"Здесь сезонные цели на общий период.\n\n" +
	"Текущий период: <b>Лето 2026</b>, до <b>01.09.2026</b>.\n\n" +
	"Лучший формат:\n" +
	"1. Направление\n" +
	"Результат: какой outcome нужен к концу периода.\n" +
	"Метрика: как понятно, что цель достигнута.\n" +
	"Еженедельный шаг: что делать каждую неделю.\n" +
	"Почему важно: зачем это держать в фокусе."

const SetupReadyText = "✅ <b>Все на месте.</b>\nТемы и стартовые сообщения уже в порядке. Ничего восстанавливать не пришлось."

const SetupRepairedText = "✨ <b>Готово!</b>\nЯ проверил пространство и восстановил все, чего не хватало.\n\n" +
	"Что дальше:\n" +
	"• в теме <b>Сегодня</b> каждый фиксирует одну цель-задачу дня;\n" +
	"• в теме <b>Рутины</b> будут daily check-in и лидерборд;\n" +
	"• в теме <b>Цели</b> будут сезонные цели и weekly review;\n" +
	"• в теме <b>Прогресс</b> будут появляться результаты задач дня."

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

func DailyTaskTextPrompt(nudge string) string {
	lines := []string{"✍️ <b>Напиши одну главную цель-задачу дня одним сообщением. Можно текстом, голосовым или медиа.</b>"}
	if nudge != "" {
		lines = append(lines, "", "💡 "+html.EscapeString(nudge))
	}
	return strings.Join(lines, "\n")
}

func DailyTaskReportPrompt(nudge string) string {
	lines := []string{"✍️ <b>Теперь напиши короткий результат одним сообщением. Можно текстом, голосовым или медиа.</b>"}
	if nudge != "" {
		lines = append(lines, "", "💡 "+html.EscapeString(nudge))
	}
	return strings.Join(lines, "\n")
}

func RoutinePlanPrompt() string {
	return "✏️ <b>Пришли список рутины одним сообщением.</b>\n\n" +
		"Каждая строка — один ежедневный пункт. Подойдут варианты:\n\n" +
		"<blockquote>зарядка\nработа\nанглийский перед сном\nйога</blockquote>\n\n" +
		"Можно с маркерами или номерами. Максимум 9 пунктов."
}

func FormatRoutineCheckinCard(checkin postgres.RoutineCheckin, displayName string, username string, notice string) string {
	person := personLabel(username, displayName)
	lines := []string{
		fmt.Sprintf("🔁 <b>Рутина</b> %s · %s", person, checkin.CheckinDate.Format("02.01")),
		"",
	}
	for _, item := range checkin.Items {
		lines = append(lines, routineItemLine(item))
		if item.ReasonText != nil && *item.ReasonText != "" {
			lines = append(lines, "   <i>что помешало:</i> "+renderInlineHTML(*item.ReasonText))
		}
	}
	if checkin.CompletedAt != nil {
		if checkin.ReflectionText != nil && *checkin.ReflectionText != "" {
			lines = append(lines, "", "<b>Итог:</b>", renderSectionHTML(*checkin.ReflectionText))
		}
		return appendNotice(lines, notice)
	}
	if nextIndex := NextRoutineItemIndex(checkin); nextIndex >= 0 {
		item := checkin.Items[nextIndex]
		lines = append(lines, "", fmt.Sprintf("<b>Пункт %d/%d:</b> %s?", nextIndex+1, len(checkin.Items), html.EscapeString(item.Text)))
	}
	return appendNotice(lines, notice)
}

func FormatRoutineReasonPrompt(checkin postgres.RoutineCheckin, itemIndex int) string {
	lines := []string{
		fmt.Sprintf("🔁 <b>Рутина</b> · %s", checkin.CheckinDate.Format("02.01")),
		"",
	}
	for _, item := range checkin.Items {
		lines = append(lines, routineItemLine(item))
	}
	if itemIndex >= 0 && itemIndex < len(checkin.Items) {
		lines = append(lines, "", fmt.Sprintf("<b>%s?</b>", html.EscapeString(checkin.Items[itemIndex].Text)), "Коротко: что помешало?")
	}
	return strings.Join(lines, "\n")
}

func FormatRoutineReflectionPrompt(checkin postgres.RoutineCheckin, displayName string, username string) string {
	lines := strings.Split(FormatRoutineCheckinCard(checkin, displayName, username, ""), "\n")
	lines = append(lines, "", "<b>Короткий итог дня:</b>", "Что помогло / что помешало / какую одну правку сделаешь завтра?")
	return strings.Join(lines, "\n")
}

func FormatRoutineLeaderboard(entries []postgres.RoutineLeaderboardEntry) string {
	lines := []string{"🏆 <b>Рутины: лидерборд</b>"}
	if len(entries) == 0 {
		return strings.Join(append(lines, "", "Пока жду первые завершенные check-in."), "\n")
	}
	limit := len(entries)
	if limit > 10 {
		limit = 10
	}
	for i := 0; i < limit; i++ {
		entry := entries[i]
		lines = append(lines, fmt.Sprintf("%d. %s — %.0f%% за 7 дней, стрик %d дней, %s", i+1, participantLabel(entry.Participant), entry.CompletionRate, entry.CurrentStreak, routineItemsCountLabel(entry.RoutineItemCount)))
	}
	best := entries[0]
	for _, entry := range entries {
		if entry.MaxStreak > best.MaxStreak {
			best = entry
		}
	}
	lines = append(lines, "", "<b>Лучший стрик сезона:</b>", fmt.Sprintf("%s — %d дней", participantLabel(best.Participant), best.MaxStreak))
	return strings.Join(lines, "\n")
}

func SeasonalGoalsPrompt() string {
	return "✏️ <b>Пришли сезонные цели одним сообщением.</b>\n\n" +
		"Текущий период: <b>Лето 2026</b>, до <b>01.09.2026</b>.\n\n" +
		"Формат для каждой цели:\n" +
		"<blockquote>1. Работа\nРезультат: получить оффер Go/backend до 01.09.2026.\nМетрика: 10 релевантных откликов или 3 касания с рынком в неделю.\nЕженедельный шаг: 2 сессии подготовки + 5 откликов.\nПочему важно: вернуть доход и закрыть финансовый провал.</blockquote>"
}

func FormatSeasonalGoalCard(goalSet postgres.SeasonalGoalSet, displayName string, username string, notice string) string {
	person := personLabel(username, displayName)
	lines := []string{
		fmt.Sprintf("🎯 <b>%s</b> %s", html.EscapeString(goalSet.PeriodTitle), person),
		fmt.Sprintf("До <b>%s</b>", goalSet.PeriodEndsOn.Format("02.01.2006")),
		"",
		renderSectionHTML(goalSet.GoalsText),
	}
	return appendNotice(lines, notice)
}

func FormatGoalWeeklyReviewPrompt(goalSet postgres.SeasonalGoalSet, displayName string, username string) string {
	person := personLabel(username, displayName)
	return strings.Join([]string{
		"🎯 <b>Еженедельная проверка целей</b>",
		"",
		fmt.Sprintf("%s, коротко ответь одним сообщением:", person),
		"",
		"1. Что сдвинулось по сезонным целям?",
		"2. Что мешало?",
		"3. Какой главный шаг на следующую неделю?",
		"",
		"<b>Твои цели:</b>",
		renderSectionHTML(goalSet.GoalsText),
	}, "\n")
}

func FormatGoalWeeklyReviewSaved(review postgres.GoalWeeklyReview) string {
	lines := []string{"✅ <b>Weekly review сохранен.</b>"}
	if review.ResponseText != nil && *review.ResponseText != "" {
		lines = append(lines, "", renderSectionHTML(*review.ResponseText))
	}
	return strings.Join(lines, "\n")
}

func FormatGoalFinalReviewPrompt(goalSet postgres.SeasonalGoalSet, displayName string, username string) string {
	person := personLabel(username, displayName)
	return strings.Join([]string{
		fmt.Sprintf("🏁 <b>Финал периода: %s</b>", html.EscapeString(goalSet.PeriodTitle)),
		"",
		fmt.Sprintf("%s, оцени сезонные цели:", person),
		"",
		renderSectionHTML(goalSet.GoalsText),
	}, "\n")
}

func FormatGoalFinalReflectionPrompt(goalSet postgres.SeasonalGoalSet, status domain.GoalFinalStatus) string {
	return strings.Join([]string{
		fmt.Sprintf("🏁 <b>Финал периода: %s</b>", html.EscapeString(goalSet.PeriodTitle)),
		"",
		"<b>Оценка:</b> " + goalFinalStatusLabel(status),
		"",
		"Теперь напиши короткий итог:",
		"• что получилось;",
		"• что не получилось;",
		"• какой вывод на следующий сезон.",
	}, "\n")
}

func FormatGoalFinalReviewSaved(goalSet postgres.SeasonalGoalSet, review postgres.GoalFinalReview) string {
	status := "—"
	if review.Status != nil {
		status = goalFinalStatusLabel(*review.Status)
	}
	lines := []string{
		fmt.Sprintf("🏁 <b>Финал периода: %s</b>", html.EscapeString(goalSet.PeriodTitle)),
		"",
		"<b>Оценка:</b> " + status,
	}
	if review.SummaryText != nil && *review.SummaryText != "" {
		lines = append(lines, "", "<b>Итог:</b>", renderSectionHTML(*review.SummaryText))
	}
	return strings.Join(lines, "\n")
}

func NextRoutineItemIndex(checkin postgres.RoutineCheckin) int {
	for i, item := range checkin.Items {
		if item.Status == nil {
			return i
		}
	}
	return -1
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
	case domain.ProgressCustomUpdate:
		return formatCustomProgressUpdate(payload)
	default:
		return "🔔 Системное сообщение\n" + fmt.Sprint(payload)
	}
}

func formatCustomProgressUpdate(payload map[string]any) string {
	title := payloadString(payload, "title")
	if title == "" {
		title = "Обновление Trackmate"
	}
	lines := []string{"🚀 <b>" + html.EscapeString(title) + "</b>"}
	if body := payloadString(payload, "body"); body != "" {
		lines = append(lines, "", html.EscapeString(body))
	}
	if items := payloadStringSlice(payload, "items"); len(items) > 0 {
		lines = append(lines, "")
		for _, item := range items {
			lines = append(lines, "• "+html.EscapeString(item))
		}
	}
	return strings.Join(lines, "\n")
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

func routineItemLine(item postgres.RoutineCheckinItem) string {
	marker := "•"
	if item.Status != nil {
		switch *item.Status {
		case domain.RoutineItemDone:
			marker = "✅"
		case domain.RoutineItemPartial:
			marker = "🔸"
		case domain.RoutineItemFailed:
			marker = "❌"
		}
	}
	return marker + " " + html.EscapeString(item.Text)
}

func goalFinalStatusLabel(status domain.GoalFinalStatus) string {
	switch status {
	case domain.GoalFinalDone:
		return "выполнены"
	case domain.GoalFinalPartial:
		return "частично"
	case domain.GoalFinalFailed:
		return "не выполнены"
	default:
		return string(status)
	}
}

func routineItemsCountLabel(count int) string {
	switch {
	case count%10 == 1 && count%100 != 11:
		return fmt.Sprintf("%d пункт", count)
	case count%10 >= 2 && count%10 <= 4 && (count%100 < 10 || count%100 >= 20):
		return fmt.Sprintf("%d пункта", count)
	default:
		return fmt.Sprintf("%d пунктов", count)
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

func participantLabel(participant postgres.Participant) string {
	username := ""
	if participant.Username != nil {
		username = *participant.Username
	}
	return personLabel(username, participant.DisplayName)
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

func renderInlineHTML(value string) string {
	if value == "" {
		return "—"
	}
	if strings.Contains(value, "<") && strings.Contains(value, ">") {
		return value
	}
	return html.EscapeString(value)
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

func payloadStringSlice(payload map[string]any, key string) []string {
	values, ok := payload[key].([]any)
	if !ok {
		if typed, ok := payload[key].([]string); ok {
			return typed
		}
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if item, ok := value.(string); ok && item != "" {
			result = append(result, item)
		}
	}
	return result
}
