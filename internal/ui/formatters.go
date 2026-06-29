package ui

import (
	"fmt"
	"html"
	"strings"

	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/storage/postgres"
)

const routineHeaderEmoji = "🌿"

const TodayControlText = "🎯 <b>Сегодня</b>\n" +
	"Здесь у каждого одна главная задача дня. Нажми кнопку ниже, чтобы зафиксировать свой главный фокус.\n\n" +
	"Как это работает:\n" +
	"— Ты формулируешь одну главную задачу дня\n" +
	"— Я закрепляю ее в отдельной карточке\n" +
	"— Вечером в этой же карточке можно отметить итог"

const ProgressIntroText = "✨ <b>Прогресс</b>\n" +
	"Здесь собирается все важное в аккуратную общую ленту.\n\n" +
	"Что появляется здесь:\n" +
	"— Выполненные задачи участников\n" +
	"— Автоматические итоги просроченных задач\n\n" +
	"Так всегда видно, кто что сделал и довел до результата."

const RoutineControlText = routineHeaderEmoji + " <b>Рутины</b>\n" +
	"Здесь живут повторяющиеся действия: зарядка, английский, йога, режим и другие ежедневные опоры.\n\n" +
	"Нажми кнопку ниже и пришли список. Я буду присылать одну карточку для отметки каждый день после 20:00.\n\n" +
	"Закрыть ее можно до 12:00 следующего дня."

const GoalsControlText = "🎯 <b>Цели</b>\n" +
	"Здесь живут долгосрочные цели, которых мы хотим достичь за сезон (например, за лето). Нажми кнопку ниже, чтобы записать свои цели.\n\n" +
	"Текущий период: <b>Лето 2026</b> (до <b>01.09.2026</b>)\n\n" +
	"Рекомендую формулировать каждую цель по такой схеме:\n" +
	"1. Направление — сфера жизни или работы (например: Спорт, Работа, Языки)\n" +
	"— <b>Результат:</b> конкретный и измеримый финал к концу сезона\n" +
	"— <b>Метрика:</b> показатель, по которому будет точно ясно, что цель достигнута\n" +
	"— <b>Шаг недели:</b> регулярное простое действие на каждую неделю\n" +
	"— <b>Зачем:</b> главный смысл цели, почему ее важно держать в фокусе"

const SetupReadyText = "✅ <b>Все на месте</b>\nТемы и стартовые сообщения в порядке"

const SetupRepairedText = "✨ <b>Готово</b>\nПространство оформлено, темы на месте\n\n" +
	"Что дальше:\n" +
	"— В <b>Сегодня</b> у каждого одна главная задача дня\n" +
	"— В <b>Рутинах</b> — ежедневные отметки и таблица результатов\n" +
	"— В <b>Целях</b> — долгосрочные цели и недельные обзоры\n" +
	"— В <b>Прогрессе</b> — общая лента выполненных задач"

func FormatSetupChecklist(ready bool, isSupergroup bool, isForum bool, isAdmin bool, canManageTopics bool, canReadMessages bool, notice string) string {
	status := "До запуска нужно закрыть несколько пунктов"
	if ready {
		status = "✅ Можно начинать: все условия выполнены"
	}
	lines := []string{
		"⚙️ <b>Подготовка пространства</b>",
		status,
		"",
		mark(isSupergroup) + " Группа переведена в супергруппу",
		mark(isForum) + " Темы включены",
		mark(isAdmin) + " Бот назначен администратором",
		mark(canManageTopics) + " Бот может управлять темами",
		mark(canReadMessages) + " Бот видит сообщения участников",
		"",
		"Когда все готово, запускай оформление группы",
	}
	return appendNotice(lines, notice)
}

func FormatDailyTaskCard(task postgres.DailyTask, displayName string, username string, notice string) string {
	person := personLabel(username, displayName)
	lines := []string{
		fmt.Sprintf("🎯 <b>Задача дня</b> %s", person),
		"",
		"<b>План:</b>",
		renderSectionHTML(task.Text),
		"",
		"<b>Состояние:</b> " + taskStatusLabel(task.Status),
	}
	if task.ReportText != nil && *task.ReportText != "" {
		lines = append(lines, "", "<b>Итог:</b>", renderSectionHTML(*task.ReportText))
	}
	return appendNotice(lines, notice)
}

func DailyTaskTextPrompt(nudge string) string {
	lines := []string{
		"✍️ <b>Напиши главную задачу дня одним сообщением</b>",
		"Можно текстом, голосом, фото или видео",
	}
	if nudge != "" {
		lines = append(lines, "", "💡 "+html.EscapeString(nudge))
	}
	return strings.Join(lines, "\n")
}

func DailyTaskReportPrompt(nudge string) string {
	lines := []string{
		"✍️ <b>Напиши короткий итог одним сообщением</b>",
		"Можно текстом, голосом, фото или видео",
	}
	if nudge != "" {
		lines = append(lines, "", "💡 "+html.EscapeString(nudge))
	}
	return strings.Join(lines, "\n")
}

func RoutinePlanPrompt() string {
	return "✏️ <b>Пришли список рутин одним сообщением</b>\n\n" +
		"Одна строка — один ежедневный пункт:\n" +
		"<blockquote>зарядка\nработа\nанглийский перед сном\nйога</blockquote>\n\n" +
		"Можно использовать маркеры или номера. Максимум 9 пунктов."
}

func FormatRoutineCheckinCard(checkin postgres.RoutineCheckin, displayName string, username string, notice string) string {
	person := personLabel(username, displayName)
	lines := []string{
		fmt.Sprintf("%s <b>Рутина за %s</b> %s", routineHeaderEmoji, checkin.CheckinDate.Format("02.01"), person),
		"Отметь, как прошел этот день",
		"",
	}
	for _, item := range checkin.Items {
		lines = append(lines, routineItemLine(item))
		if item.ReasonText != nil && *item.ReasonText != "" {
			lines = append(lines, "   <i>причина:</i> "+renderInlineHTML(*item.ReasonText))
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
		lines = append(lines, "", fmt.Sprintf("<b>%d/%d:</b> %s?", nextIndex+1, len(checkin.Items), html.EscapeString(item.Text)))
	}
	return appendNotice(lines, notice)
}

func FormatRoutineReasonPrompt(checkin postgres.RoutineCheckin, displayName string, username string, itemIndex int) string {
	lines := strings.Split(FormatRoutineCheckinCard(checkin, displayName, username, ""), "\n")
	if itemIndex >= 0 && itemIndex < len(checkin.Items) {
		lines = append(lines, "Что помешало?")
	}
	return strings.Join(lines, "\n")
}

func FormatRoutineReflectionPrompt(checkin postgres.RoutineCheckin, displayName string, username string) string {
	lines := strings.Split(FormatRoutineCheckinCard(checkin, displayName, username, ""), "\n")
	lines = append(lines, "", "<b>Итог дня</b>", "Что помогло? Что мешало? Что изменишь завтра?")
	return strings.Join(lines, "\n")
}

func RoutineReminderText(checkin postgres.RoutineCheckin) string {
	return strings.Join([]string{
		fmt.Sprintf("🔔 <b>Рутина за %s еще не закрыта</b>", checkin.CheckinDate.Format("02.01")),
		"Закрой ее до 12:00, иначе неотмеченные пункты будут засчитаны как невыполненные.",
	}, "\n")
}

func RoutineAutoClosedText(checkin postgres.RoutineCheckin) string {
	return strings.Join([]string{
		"⏰ <b>Время вышло</b>",
		fmt.Sprintf("Рутина за %s закрыта автоматически: неотмеченные пункты засчитаны как невыполненные.", checkin.CheckinDate.Format("02.01")),
	}, "\n")
}

func FormatRoutineLeaderboard(entries []postgres.RoutineLeaderboardEntry) string {
	lines := []string{"🏆 <b>Таблица рутин</b>"}
	if len(entries) == 0 {
		return strings.Join(append(lines, "", "Пока жду первые завершенные проверки"), "\n")
	}
	limit := len(entries)
	if limit > 10 {
		limit = 10
	}
	for i := 0; i < limit; i++ {
		entry := entries[i]
		lines = append(lines, fmt.Sprintf("%d. %s — %.0f%% за 7 дней, серия %d дней, %s", i+1, participantLabel(entry.Participant), entry.CompletionRate, entry.CurrentStreak, routineItemsCountLabel(entry.RoutineItemCount)))
	}
	best := entries[0]
	for _, entry := range entries {
		if entry.MaxStreak > best.MaxStreak {
			best = entry
		}
	}
	lines = append(lines, "", "<b>Лучшая серия сезона</b>", fmt.Sprintf("%s — %d дней", participantLabel(best.Participant), best.MaxStreak))
	return strings.Join(lines, "\n")
}

func SeasonalGoalsPrompt() string {
	return "✏️ <b>Пришли сезонные цели одним сообщением</b>\n\n" +
		"Текущий период: <b>Лето 2026</b> (до <b>01.09.2026</b>)\n\n" +
		"Используй для каждой цели эту схему:\n" +
		"<blockquote>1. Направление (например, Работа)\n— Результат: конкретный измеримый итог к концу сезона\n— Метрика: как именно ты измеришь успех\n— Шаг недели: что делать каждую неделю\n— Зачем: почему эта цель важна для тебя</blockquote>"
}

func FormatSeasonalGoalCard(goalSet postgres.SeasonalGoalSet, displayName string, username string, notice string) string {
	person := personLabel(username, displayName)
	lines := []string{
		fmt.Sprintf("🎯 <b>Цели на %s</b> · %s", html.EscapeString(strings.ToLower(goalSet.PeriodTitle)), person),
		fmt.Sprintf("До <b>%s</b>", goalSet.PeriodEndsOn.Format("02.01.2006")),
		"",
		renderSectionHTML(goalSet.GoalsText),
	}
	return appendNotice(lines, notice)
}

func FormatGoalWeeklyReviewPrompt(goalSet postgres.SeasonalGoalSet, displayName string, username string) string {
	person := personLabel(username, displayName)
	return strings.Join([]string{
		"🎯 <b>Недельный обзор целей</b>",
		"",
		fmt.Sprintf("%s, ответь одним сообщением на три вопроса:", person),
		"",
		"1. Какие шаги сделаны по сезонным целям за эту неделю?",
		"2. С какими сложностями пришлось столкнуться?",
		"3. Какой главный шаг планируешь на следующую неделю?",
		"",
		"<b>Твои цели:</b>",
		renderSectionHTML(goalSet.GoalsText),
	}, "\n")
}

func FormatGoalWeeklyReviewSaved(review postgres.GoalWeeklyReview) string {
	lines := []string{"✅ <b>Недельный обзор сохранен</b>"}
	if review.ResponseText != nil && *review.ResponseText != "" {
		lines = append(lines, "", renderSectionHTML(*review.ResponseText))
	}
	return strings.Join(lines, "\n")
}

func FormatGoalFinalReviewPrompt(goalSet postgres.SeasonalGoalSet, displayName string, username string) string {
	person := personLabel(username, displayName)
	return strings.Join([]string{
		fmt.Sprintf("🏁 <b>Итог периода: %s</b>", html.EscapeString(goalSet.PeriodTitle)),
		"",
		fmt.Sprintf("%s, оцени сезонные цели:", person),
		"",
		renderSectionHTML(goalSet.GoalsText),
	}, "\n")
}

func FormatGoalFinalReflectionPrompt(goalSet postgres.SeasonalGoalSet, status domain.GoalFinalStatus) string {
	return strings.Join([]string{
		fmt.Sprintf("🏁 <b>Итог периода: %s</b>", html.EscapeString(goalSet.PeriodTitle)),
		"",
		"<b>Оценка:</b> " + goalFinalStatusLabel(status),
		"",
		"Опиши конкретные результаты по целям:",
		"— Что именно удалось довести до конца",
		"— Что осталось невыполненным и почему",
		"— Какие выводы и задачи переносишь на следующий сезон",
	}, "\n")
}

func FormatGoalFinalReviewSaved(goalSet postgres.SeasonalGoalSet, review postgres.GoalFinalReview) string {
	status := "—"
	if review.Status != nil {
		status = goalFinalStatusLabel(*review.Status)
	}
	lines := []string{
		fmt.Sprintf("🏁 <b>Итог периода: %s</b>", html.EscapeString(goalSet.PeriodTitle)),
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
			"<b>План:</b>",
			renderSectionHTML(payloadString(payload, "task_html")),
			"",
			"<b>Итог:</b>",
			renderSectionHTML(payloadString(payload, "report_html")),
		}, "\n")
	case domain.ProgressDailyTaskAutoFail:
		task := payloadLink(payload, "task_link", "задачу дня")
		return strings.Join([]string{
			fmt.Sprintf("⏰ <b>%s не выполнил %s вовремя</b>", person, task),
			"",
			"<b>План:</b>",
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
			lines = append(lines, "— "+html.EscapeString(item))
		}
	}
	return strings.Join(lines, "\n")
}

func AlertText(kind domain.AlertKind) string {
	if kind == domain.AlertDayClosedPendingReport {
		return "🔔 День закончился, а итог по задаче еще не подведен"
	}
	return "⏰ Время вышло. Задача отмечена как не выполненная"
}

func mark(value bool) string {
	if value {
		return "✅"
	}
	return "—"
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
		return "ждет итога"
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
	marker := "—"
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
