package ui

import (
	"fmt"
	"html"
	"strings"

	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/messages"
	"github.com/igor/trackmate/internal/storage/postgres"
)

var (
	TodayControlText   = messages.Text("today.control")
	ProgressIntroText  = messages.Text("progress.intro")
	RoutineControlText = messages.Text("routine.control")
	GoalsControlText   = messages.Text("goals.control")
	SetupReadyText     = messages.Text("setup.ready")
	SetupRepairedText  = messages.Text("setup.repaired")
	routineHeaderEmoji = messages.Text("routine.header_emoji")
)

func FormatSetupChecklist(ready bool, isSupergroup bool, isForum bool, isAdmin bool, canManageTopics bool, canReadMessages bool, notice string) string {
	status := messages.Text("setup.checklist.pending")
	if ready {
		status = messages.Text("setup.checklist.ready")
	}
	lines := []string{
		messages.Text("setup.checklist.title"),
		status,
		"",
		mark(isSupergroup) + " " + messages.Text("setup.checklist.supergroup"),
		mark(isForum) + " " + messages.Text("setup.checklist.forum"),
		mark(isAdmin) + " " + messages.Text("setup.checklist.admin"),
		mark(canManageTopics) + " " + messages.Text("setup.checklist.topics"),
		mark(canReadMessages) + " " + messages.Text("setup.checklist.messages"),
		"",
		messages.Text("setup.checklist.footer"),
	}
	return appendNotice(lines, notice)
}

func FormatDailyTaskCard(task postgres.DailyTask, displayName string, username string, notice string) string {
	person := personLabel(username, displayName)
	lines := []string{
		messages.Format("daily.card.title", "person", person),
		"",
		messages.Text("daily.card.plan"),
		renderSectionHTML(task.Text),
		"",
		messages.Format("daily.card.status", "status", taskStatusLabel(task.Status)),
	}
	if task.ReportText != nil && *task.ReportText != "" {
		lines = append(lines, "", messages.Text("daily.card.report"), renderSectionHTML(*task.ReportText))
	}
	return appendNotice(lines, notice)
}

func DailyTaskTextPrompt(nudge string) string {
	lines := strings.Split(messages.Text("daily.prompt.task"), "\n")
	if nudge != "" {
		lines = append(lines, "", "💡 "+html.EscapeString(nudge))
	}
	return strings.Join(lines, "\n")
}

func DailyTaskReportPrompt(nudge string) string {
	lines := strings.Split(messages.Text("daily.prompt.report"), "\n")
	if nudge != "" {
		lines = append(lines, "", "💡 "+html.EscapeString(nudge))
	}
	return strings.Join(lines, "\n")
}

func RoutinePlanPrompt() string {
	return messages.Text("routine.plan.prompt")
}

func FormatRoutineCheckinCard(checkin postgres.RoutineCheckin, displayName string, username string, notice string) string {
	return formatRoutineCheckinCard(checkin, displayName, username, notice)
}

func FormatRoutineCheckinStatusCard(checkin postgres.RoutineCheckin, displayName string, username string, notice string) string {
	return formatRoutineCheckinCard(checkin, displayName, username, notice)
}

func formatRoutineCheckinCard(checkin postgres.RoutineCheckin, displayName string, username string, notice string) string {
	person := personLabel(username, displayName)
	lines := []string{
		messages.Format("routine.card.title", "emoji", routineHeaderEmoji, "date", checkin.CheckinDate.Format("02.01"), "person", person),
		messages.Text("routine.card.subtitle"),
		"",
	}
	for _, item := range checkin.Items {
		lines = append(lines, routineItemLine(item))
		if item.ReasonText != nil && *item.ReasonText != "" {
			lines = append(lines, messages.Format("routine.item.reason_label", "reason", renderInlineHTML(*item.ReasonText)))
		}
	}
	if checkin.CompletedAt != nil {
		if checkin.ReflectionText != nil && *checkin.ReflectionText != "" {
			lines = append(lines, "", messages.Text("daily.card.report"), renderSectionHTML(*checkin.ReflectionText))
		}
		return appendNotice(lines, notice)
	}
	return appendNotice(lines, notice)
}

func FormatRoutineReasonPrompt(itemText string) string {
	return messages.Format("routine.reason.prompt", "item", html.EscapeString(itemText))
}

func RoutineReminderText(checkin postgres.RoutineCheckin, displayName string, username string, userID int64) string {
	person := userLinkLabel(displayName, username, userID)
	if domain.ShouldShowPersonalAlert(username, fmt.Sprintf("routine-reminder:%d", checkin.ID)) {
		return messages.Format(
			"routine.reminder.egor",
			"date", checkin.CheckinDate.Format("02.01"),
			"person", person,
		)
	}
	return messages.Format(
		"routine.reminder",
		"date", checkin.CheckinDate.Format("02.01"),
		"person", person,
	)
}

func RoutineAutoClosedText(checkin postgres.RoutineCheckin, displayName string, username string, userID int64) string {
	if domain.ShouldShowPersonalAlert(username, fmt.Sprintf("routine-auto-closed:%d", checkin.ID)) {
		return messages.Format(
			"routine.auto_closed.egor",
			"date", checkin.CheckinDate.Format("02.01"),
			"person", userLinkLabel(displayName, username, userID),
		)
	}
	return messages.Format("routine.auto_closed", "date", checkin.CheckinDate.Format("02.01"))
}

func FormatRoutineLeaderboard(entries []postgres.RoutineLeaderboardEntry) string {
	lines := []string{messages.Text("routine.leaderboard.title")}
	if len(entries) == 0 {
		return strings.Join(append(lines, "", messages.Text("routine.leaderboard.empty")), "\n")
	}
	limit := len(entries)
	if limit > 10 {
		limit = 10
	}
	for i := 0; i < limit; i++ {
		entry := entries[i]
		lines = append(lines, messages.Format(
			"routine.leaderboard.entry",
			"rank", fmt.Sprint(i+1),
			"participant", participantLabel(entry.Participant),
			"rate", fmt.Sprintf("%.0f", entry.CompletionRate),
			"streak", fmt.Sprint(entry.CurrentStreak),
			"items", routineItemsCountLabel(entry.RoutineItemCount),
		))
	}
	best := entries[0]
	for _, entry := range entries {
		if entry.MaxStreak > best.MaxStreak {
			best = entry
		}
	}
	lines = append(lines, "", messages.Text("routine.leaderboard.best_title"), messages.Format(
		"routine.leaderboard.best_entry",
		"participant", participantLabel(best.Participant),
		"streak", fmt.Sprint(best.MaxStreak),
	))
	return strings.Join(lines, "\n")
}

func SeasonalGoalsPrompt() string {
	return messages.Text("goals.prompt")
}

func FormatGoalsSaved(goalsLink string) string {
	if strings.TrimSpace(goalsLink) == "" {
		return messages.Text("goals.saved_no_link")
	}
	return messages.Format("goals.saved", "link", html.EscapeString(goalsLink))
}

func FormatSeasonalGoalCard(goalSet postgres.SeasonalGoalSet, displayName string, username string, notice string) string {
	person := personLabel(username, displayName)
	lines := []string{
		messages.Format("goals.card.title", "period", html.EscapeString(strings.ToLower(goalSet.PeriodTitle)), "person", person),
		messages.Format("goals.card.deadline", "date", goalSet.PeriodEndsOn.Format("02.01.2006")),
		"",
		renderSectionHTML(goalSet.GoalsText),
	}
	return appendNotice(lines, notice)
}

func FormatGoalWeeklyReviewPrompt(goalSet postgres.SeasonalGoalSet, displayName string, username string, goalsLink string, daysLeft int, reviewsLeft int) string {
	person := personLabel(username, displayName)
	lines := []string{
		messages.Text("goals.weekly.title"),
		"",
		messages.Format("goals.weekly.intro", "person", person),
		"",
	}
	if goalsLink != "" {
		lines = append(lines, messages.Format("goals.weekly.source", "link", html.EscapeString(goalsLink)))
	} else {
		lines = append(lines, messages.Text("goals.weekly.source_missing"))
	}
	lines = append(lines,
		messages.Format("goals.weekly.countdown", "days", daysLabel(daysLeft), "reviews", reviewsLabel(reviewsLeft)),
		"",
		messages.Text("goals.weekly.q1"),
		messages.Text("goals.weekly.q2"),
		messages.Text("goals.weekly.q3"),
	)
	return strings.Join(lines, "\n")
}

func FormatGoalFinalReviewPrompt(goalSet postgres.SeasonalGoalSet, displayName string, username string, goalsLink string) string {
	person := personLabel(username, displayName)
	lines := []string{
		messages.Format("goals.final.title", "period", html.EscapeString(goalSet.PeriodTitle)),
		"",
		messages.Format("goals.final.ask_status", "person", person),
	}
	if goalsLink != "" {
		lines = append(lines, "", messages.Format("goals.weekly.source", "link", html.EscapeString(goalsLink)))
	} else {
		lines = append(lines, "", messages.Text("goals.weekly.source_missing"))
	}
	return strings.Join(lines, "\n")
}

func daysLabel(days int) string {
	if days < 0 {
		days = 0
	}
	return fmt.Sprintf("%d %s", days, russianPlural(days, messages.Text("goals.days.one"), messages.Text("goals.days.few"), messages.Text("goals.days.many")))
}

func reviewsLabel(reviews int) string {
	if reviews < 0 {
		reviews = 0
	}
	return fmt.Sprintf("%d %s", reviews, russianPlural(reviews, messages.Text("goals.reviews.one"), messages.Text("goals.reviews.few"), messages.Text("goals.reviews.many")))
}

func russianPlural(count int, one string, few string, many string) string {
	mod100 := count % 100
	if mod100 >= 11 && mod100 <= 14 {
		return many
	}
	switch count % 10 {
	case 1:
		return one
	case 2, 3, 4:
		return few
	default:
		return many
	}
}

func FormatGoalWeeklyReviewSaved(review postgres.GoalWeeklyReview) string {
	lines := []string{messages.Text("goals.weekly.saved")}
	if review.ResponseText != nil && *review.ResponseText != "" {
		lines = append(lines, "", renderSectionHTML(*review.ResponseText))
	}
	return strings.Join(lines, "\n")
}

func FormatGoalFinalReflectionPrompt(goalSet postgres.SeasonalGoalSet, status domain.GoalFinalStatus) string {
	return strings.Join([]string{
		messages.Format("goals.final.title", "period", html.EscapeString(goalSet.PeriodTitle)),
		"",
		messages.Format("goals.final.score", "status", goalFinalStatusLabel(status)),
		"",
		messages.Text("goals.final.reflection_intro"),
		messages.Text("goals.final.reflection_done"),
		messages.Text("goals.final.reflection_failed"),
		messages.Text("goals.final.reflection_next"),
	}, "\n")
}

func FormatGoalFinalReviewSaved(goalSet postgres.SeasonalGoalSet, review postgres.GoalFinalReview) string {
	status := "—"
	if review.Status != nil {
		status = goalFinalStatusLabel(*review.Status)
	}
	lines := []string{
		messages.Format("goals.final.title", "period", html.EscapeString(goalSet.PeriodTitle)),
		"",
		messages.Format("goals.final.score", "status", status),
	}
	if review.SummaryText != nil && *review.SummaryText != "" {
		lines = append(lines, "", messages.Text("goals.final.saved_summary"), renderSectionHTML(*review.SummaryText))
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
		taskLabel := messages.Text("progress.daily.task_link")
		task := payloadLink(payload, "task_link", taskLabel)
		action := dailyTaskClosedAction(payloadString(payload, "status"), payloadString(payload, "report_link"))
		title := strings.Replace(dailyTaskClosedTitle(payloadString(payload, "status"), person, action), taskLabel, task, 1)
		return strings.Join([]string{
			title,
			"",
			messages.Text("daily.card.plan"),
			renderSectionHTML(payloadString(payload, "task_html")),
			"",
			messages.Text("daily.card.report"),
			renderSectionHTML(linkedReportHTML(payload)),
		}, "\n")
	case domain.ProgressDailyTaskAutoFail:
		task := payloadLink(payload, "task_link", messages.Text("progress.daily.task_link"))
		return strings.Join([]string{
			messages.Format("progress.daily.auto_failed", "person", person, "task", task),
			"",
			messages.Text("daily.card.plan"),
			renderSectionHTML(payloadString(payload, "task_html")),
		}, "\n")
	case domain.ProgressCustomUpdate:
		return formatCustomProgressUpdate(payload)
	default:
		return messages.Text("progress.system") + "\n" + fmt.Sprint(payload)
	}
}

func formatCustomProgressUpdate(payload map[string]any) string {
	title := payloadString(payload, "title")
	if title == "" {
		title = messages.Text("progress.custom.default_title")
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

func AlertText(kind domain.AlertKind, displayName string, username string, userID int64, seed string) string {
	if domain.ShouldShowPersonalAlert(username, seed) {
		return messages.Format("alert.egor", "person", userLinkLabel(displayName, username, userID))
	}
	if kind == domain.AlertDayClosedPendingReport {
		return messages.Text("alert.day_closed_pending_report")
	}
	return messages.Text("alert.auto_failed")
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
		return messages.Text("daily.status.active")
	case domain.DailyTaskAwaitingReport:
		return messages.Text("daily.status.awaiting_report")
	case domain.DailyTaskDone:
		return messages.Text("daily.status.done")
	case domain.DailyTaskPartial:
		return messages.Text("daily.status.partial")
	case domain.DailyTaskFailed:
		return messages.Text("daily.status.failed")
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
		return messages.Text("goals.status.done")
	case domain.GoalFinalPartial:
		return messages.Text("goals.status.partial")
	case domain.GoalFinalFailed:
		return messages.Text("goals.status.failed")
	default:
		return string(status)
	}
}

func routineItemsCountLabel(count int) string {
	switch {
	case count%10 == 1 && count%100 != 11:
		return messages.Format("routine.items_count.one", "count", fmt.Sprint(count))
	case count%10 >= 2 && count%10 <= 4 && (count%100 < 10 || count%100 >= 20):
		return messages.Format("routine.items_count.few", "count", fmt.Sprint(count))
	default:
		return messages.Format("routine.items_count.many", "count", fmt.Sprint(count))
	}
}

func dailyTaskClosedTitle(status string, person string, action string) string {
	switch status {
	case "done":
		return messages.Format("progress.daily.closed.done", "person", person, "action", action)
	case "partial":
		return messages.Format("progress.daily.closed.partial", "person", person, "action", action)
	case "failed":
		return messages.Format("progress.daily.closed.failed", "person", person, "action", action)
	default:
		return messages.Format("progress.daily.closed.default", "person", person, "action", action)
	}
}

func dailyTaskClosedAction(status string, reportLink string) string {
	label := messages.Text("progress.daily.action.done")
	switch status {
	case "partial":
		label = messages.Text("progress.daily.action.partial")
	case "failed":
		label = messages.Text("progress.daily.action.failed")
	}
	if reportLink == "" {
		return label
	}
	return fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(reportLink), label)
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
		displayName = messages.Text("participant.fallback_name")
	}
	if username := payloadString(payload, "username"); username != "" {
		return userLinkLabel(displayName, username, payloadInt64(payload, "user_id"))
	}
	return userLinkLabel(displayName, "", payloadInt64(payload, "user_id"))
}

func userLinkLabel(displayName string, username string, userID int64) string {
	label := displayName
	if strings.TrimSpace(label) == "" {
		label = username
	}
	if strings.TrimSpace(label) == "" {
		label = messages.Text("participant.fallback_name")
	}
	escaped := html.EscapeString(label)
	if userID == 0 {
		return escaped
	}
	return fmt.Sprintf(`<a href="tg://user?id=%d">%s</a>`, userID, escaped)
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

func linkedReportHTML(payload map[string]any) string {
	report := payloadString(payload, "report_html")
	reportLink := payloadString(payload, "report_link")
	if report == "" || reportLink == "" || strings.Contains(report, "<") || strings.Contains(report, ">") {
		return report
	}
	return fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(reportLink), report)
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
