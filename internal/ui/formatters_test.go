package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/storage/postgres"
)

func TestFormatProgressEventDailyTaskClosed(t *testing.T) {
	event := postgres.ProgressEvent{
		EventType: domain.ProgressDailyTaskClosed,
		Payload: map[string]any{
			"user_id":      float64(42),
			"display_name": "Igor",
			"username":     "igor",
			"status":       "done",
			"task_html":    `<a href="https://example.com/task">Написать движок на Go</a>`,
			"task_link":    "https://t.me/c/1/300?thread=10",
			"report_html":  "Готово",
			"report_link":  "https://t.me/c/1/301?thread=10",
		},
	}
	got := FormatProgressEvent(event)
	for _, part := range []string{
		`<a href="tg://user?id=42">Igor</a>`,
		`<a href="https://t.me/c/1/301?thread=10">выполнил</a> <a href="https://t.me/c/1/300?thread=10">задачу дня</a>`,
		"Написать движок на Go",
		"<blockquote>Готово</blockquote>",
	} {
		if !strings.Contains(got, part) {
			t.Fatalf("formatted task event missing %q: %s", part, got)
		}
	}
	if strings.Contains(got, `<a href="https://t.me/c/1/301?thread=10">Готово</a>`) {
		t.Fatalf("formatted task event should not link report text: %s", got)
	}
	if strings.Contains(got, "<b>Задача:</b>\n\n<blockquote>") || strings.Contains(got, "<b>Результат:</b>\n\n<blockquote>") {
		t.Fatalf("formatted task event has extra blank line around blockquote: %s", got)
	}
	if strings.Contains(got, "https://example.com/task") {
		t.Fatalf("formatted task event should not link the task text: %s", got)
	}
}

func TestFormatProgressEventDailyTaskAutoFailedDoesNotLinkTaskText(t *testing.T) {
	event := postgres.ProgressEvent{
		EventType: domain.ProgressDailyTaskAutoFail,
		Payload: map[string]any{
			"user_id":      float64(42),
			"display_name": "Igor",
			"task_html":    `<a href="https://example.com/task">Написать движок на Go</a>`,
			"task_link":    "https://t.me/c/1/300?thread=10",
		},
	}
	got := FormatProgressEvent(event)
	if !strings.Contains(got, `<a href="https://t.me/c/1/300?thread=10">задачу дня</a>`) {
		t.Fatalf("formatted auto-fail event should link the task label: %s", got)
	}
	if strings.Contains(got, "https://example.com/task") {
		t.Fatalf("formatted auto-fail event should not link the task text: %s", got)
	}
}

func TestFormatProgressEventDailyTaskPartialUsesActionPhrase(t *testing.T) {
	event := postgres.ProgressEvent{
		EventType: domain.ProgressDailyTaskClosed,
		Payload: map[string]any{
			"user_id":      float64(42),
			"display_name": "Игорь",
			"username":     "igor",
			"status":       "partial",
			"task_html":    "Разобрать задачу",
			"task_link":    "https://t.me/c/1/300?thread=10",
			"report_html":  "Сделал половину",
			"report_link":  "https://t.me/c/1/301?thread=10",
		},
	}
	got := FormatProgressEvent(event)
	if !strings.Contains(got, `частично выполнил</a> <a href="https://t.me/c/1/300?thread=10">задачу дня</a>`) {
		t.Fatalf("formatted partial task event should use action phrase: %s", got)
	}
	if strings.Contains(got, "закрыл задачу дня") {
		t.Fatalf("formatted partial task event should not use old wording: %s", got)
	}
}

func TestFormatDailyTaskCardShowsPlanWithoutExtraBlockquoteGap(t *testing.T) {
	task := postgres.DailyTask{
		OwnerUserID: 42,
		Text:        `<a href="https://example.com/task">Подготовить результат по задаче</a>`,
		Status:      domain.DailyTaskActive,
	}
	got := FormatDailyTaskCard(task, "Игорь", "igor", "")
	for _, part := range []string{`🎯 <b>Задача дня</b> <a href="tg://user?id=42">Игорь</a>`, "<b>Задача:</b>", "Подготовить результат по задаче"} {
		if !strings.Contains(got, part) {
			t.Fatalf("daily task card missing %q: %s", part, got)
		}
	}
	for _, forbidden := range []string{"<b>Состояние:</b>", "<b>Статус:</b>", "<b>План:</b>"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("daily task card should not contain %q: %s", forbidden, got)
		}
	}
	if strings.Contains(got, "<b>Задача:</b>\n\n<blockquote>") {
		t.Fatalf("daily task card has extra blank line before plan blockquote: %s", got)
	}
	if strings.Contains(got, "https://example.com/task") {
		t.Fatalf("daily task card should not link task text: %s", got)
	}

	done := task
	done.Status = domain.DailyTaskDone
	report := "Результат готов"
	done.ReportText = &report
	closed := FormatDailyTaskCard(done, "Игорь", "igor", "")
	for _, part := range []string{`✅ <b><a href="tg://user?id=42">Игорь</a> выполнил задачу дня</b>`, "<b>Результат:</b>", "Результат готов"} {
		if !strings.Contains(closed, part) {
			t.Fatalf("closed daily task card missing %q: %s", part, closed)
		}
	}
	if strings.Contains(closed, "@igor") {
		t.Fatalf("closed daily task card should use display name instead of username mention: %s", closed)
	}
	if strings.Contains(closed, "https://example.com/task") {
		t.Fatalf("closed daily task card should not link task text: %s", closed)
	}
	if strings.Contains(closed, "<b>Состояние:</b>") || strings.Contains(closed, "<b>Статус:</b>") {
		t.Fatalf("closed daily task card should encode status in title only: %s", closed)
	}

	partial := task
	partial.Status = domain.DailyTaskPartial
	partial.ReportText = &report
	if got := FormatDailyTaskCard(partial, "Игорь", "igor", ""); !strings.Contains(got, `🔸 <b><a href="tg://user?id=42">Игорь</a> частично выполнил задачу дня</b>`) {
		t.Fatalf("partial daily task card title mismatch: %s", got)
	}

	failed := task
	failed.Status = domain.DailyTaskFailed
	failed.ReportText = &report
	if got := FormatDailyTaskCard(failed, "Игорь", "igor", ""); !strings.Contains(got, `❌ <b><a href="tg://user?id=42">Игорь</a> не выполнил задачу дня</b>`) {
		t.Fatalf("failed daily task card title mismatch: %s", got)
	}
}

func TestGeneratedMultilineMessagesKeepHeaderGap(t *testing.T) {
	checklist := FormatSetupChecklist(true, true, true, true, true, true, "")
	if !strings.Contains(checklist, "⚙️ <b>Подготовка пространства</b>\n\n✅ Можно начинать") {
		t.Fatalf("setup checklist should keep header gap: %s", checklist)
	}

	checkin := postgres.RoutineCheckin{
		CheckinDate: time.Date(2026, 6, 24, 0, 0, 0, 0, time.UTC),
		Items:       []postgres.RoutineCheckinItem{{Text: "зарядка"}},
	}
	routineCard := FormatRoutineCheckinCard(checkin, "Игорь", "igor", "")
	if !strings.Contains(routineCard, "🌿 <b>Рутина за 24.06</b> @igor\n\nОтметь пункты за этот день") {
		t.Fatalf("routine card should keep header gap: %s", routineCard)
	}

	username := "igor"
	leaderboard := FormatRoutineLeaderboard([]postgres.RoutineLeaderboardEntry{{
		Participant: postgres.Participant{
			Username:    &username,
			DisplayName: "Igor",
		},
		RoutineItemCount: 1,
	}})
	if !strings.Contains(leaderboard, "🏆 <b>Таблица рутин</b>\n\n1. @igor") {
		t.Fatalf("routine leaderboard should keep header gap: %s", leaderboard)
	}

	goalCard := FormatSeasonalGoalCard(postgres.SeasonalGoalSet{
		PeriodTitle:  "Лето 2026",
		PeriodEndsOn: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		GoalsText:    "1. Работа",
	}, "Игорь", "igor", "")
	if !strings.Contains(goalCard, "🎯 <b>Цели на лето 2026</b> · @igor\n\nДо <b>01.09.2026</b>") {
		t.Fatalf("goals card should keep header gap: %s", goalCard)
	}
}

func TestFormatProgressEventCustomUpdate(t *testing.T) {
	event := postgres.ProgressEvent{
		EventType: domain.ProgressCustomUpdate,
		Payload: map[string]any{
			"title": "Встречайте: Trackmate 1.0 на Go",
			"body":  "Бот переехал на новый движок.",
			"items": []any{
				"сохранили задачи, итоги и прогресс",
				"удалили старые материалы",
			},
		},
	}
	got := FormatProgressEvent(event)
	for _, part := range []string{"Встречайте: Trackmate 1.0 на Go", "новый движок", "сохранили задачи", "сохранили задачи, итоги", "удалили старые материалы"} {
		if !strings.Contains(got, part) {
			t.Fatalf("formatted custom update missing %q: %s", part, got)
		}
	}
}

func TestFormatRoutineLeaderboardShowsRateSeriesAndItemCount(t *testing.T) {
	username := "igor"
	got := FormatRoutineLeaderboard([]postgres.RoutineLeaderboardEntry{{
		Participant: postgres.Participant{
			Username:    &username,
			DisplayName: "Igor",
		},
		CompletionRate:   92,
		CurrentStreak:    5,
		MaxStreak:        9,
		RoutineItemCount: 4,
	}})
	for _, part := range []string{"Таблица рутин", "92% за 7 дней", "серия 5 дней", "4 пункта", "Лучшая серия сезона"} {
		if !strings.Contains(got, part) {
			t.Fatalf("routine table missing %q: %s", part, got)
		}
	}
}

func TestFormatRoutineCheckinCardClarifiesDateScope(t *testing.T) {
	checkin := postgres.RoutineCheckin{
		CheckinDate: time.Date(2026, 6, 24, 0, 0, 0, 0, time.UTC),
		Items: []postgres.RoutineCheckinItem{
			{Text: "зарядка"},
			{Text: "английский"},
		},
	}
	card := FormatRoutineCheckinCard(checkin, "Игорь", "igor", "")
	for _, part := range []string{"🌿 <b>Рутина за 24.06</b> @igor", "Отметь пункты за этот день", "— зарядка", "— английский"} {
		if !strings.Contains(card, part) {
			t.Fatalf("routine card missing %q: %s", part, card)
		}
	}
	if strings.Contains(card, "1/2:") || strings.Contains(card, "зарядка?") {
		t.Fatalf("routine card should not ask item inside the main card: %s", card)
	}
	statusOnly := FormatRoutineCheckinStatusCard(checkin, "Игорь", "igor", "")
	if strings.Contains(statusOnly, "<b>1/2:</b> зарядка?") {
		t.Fatalf("routine status card should not ask the next item: %s", statusOnly)
	}
	reason := FormatRoutineReasonPrompt("зарядка")
	for _, part := range []string{"Что помешало?", "зарядка"} {
		if !strings.Contains(reason, part) {
			t.Fatalf("routine reason prompt missing %q: %s", part, reason)
		}
	}
}

func TestFinalCopyUsesCalmStyleAndDashLists(t *testing.T) {
	if !strings.Contains(TodayControlText, "Здесь у каждого одна главная задача дня") {
		t.Fatalf("today control text should keep old topic style: %s", TodayControlText)
	}
	if !strings.Contains(RoutineControlText, "Здесь живет одна ежедневная рутина") {
		t.Fatalf("routine control text should keep old topic style: %s", RoutineControlText)
	}
	if !strings.Contains(ProgressIntroText, "Здесь собирается все важное") {
		t.Fatalf("progress intro should keep old topic style: %s", ProgressIntroText)
	}
	if !strings.Contains(GoalsControlText, "долгосрочные цели, которых мы хотим достичь за сезон") {
		t.Fatalf("goals control should explain long-term seasonal goals: %s", GoalsControlText)
	}
	for name, text := range map[string]string{
		"today":    TodayControlText,
		"progress": ProgressIntroText,
		"routine":  RoutineControlText,
		"goals":    GoalsControlText,
		"prompt":   SeasonalGoalsPrompt(),
	} {
		if strings.Contains(text, "•") {
			t.Fatalf("%s text should use dashes instead of bullet markers: %s", name, text)
		}
	}

	goalSet := postgres.SeasonalGoalSet{PeriodTitle: "Лето 2026", GoalsText: "1. Работа"}
	weekly := FormatGoalWeeklyReviewPrompt(goalSet, "Игорь", "igor", "https://t.me/c/1/301?thread=40", 65, 4)
	for _, part := range []string{
		"Вопросы по целям",
		`<a href="https://t.me/c/1/301?thread=40">открыть список</a>`,
		"До итога:</b> 65 дней · 4 проверки",
		"Что продвинулось за последние две недели?",
		"Что мешало?",
		"Что сделаешь в следующие две недели?",
	} {
		if !strings.Contains(weekly, part) {
			t.Fatalf("weekly prompt missing %q: %s", part, weekly)
		}
	}
	if strings.Contains(weekly, "1. Работа") || strings.Contains(weekly, "сезонным") {
		t.Fatalf("weekly prompt should not echo full goals or say seasonal: %s", weekly)
	}

	finalPrompt := FormatGoalFinalReviewPrompt(goalSet, "Игорь", "igor", "https://t.me/c/1/301?thread=40")
	if !strings.Contains(finalPrompt, `<a href="https://t.me/c/1/301?thread=40">открыть список</a>`) || strings.Contains(finalPrompt, "1. Работа") {
		t.Fatalf("final prompt should link to goals without echoing them: %s", finalPrompt)
	}

	saved := FormatGoalsSaved("https://t.me/c/1/301?thread=40")
	if !strings.Contains(saved, `<a href="https://t.me/c/1/301?thread=40">Цели</a> записаны`) {
		t.Fatalf("goals saved confirmation should link the title word: %s", saved)
	}

	final := FormatGoalFinalReflectionPrompt(goalSet, domain.GoalFinalPartial)
	for _, part := range []string{
		"Опиши конкретные результаты по целям:",
		"— Что именно удалось довести до конца",
		"— Что осталось невыполненным и почему",
		"— Какие выводы и задачи переносишь на следующий сезон",
	} {
		if !strings.Contains(final, part) {
			t.Fatalf("final reflection prompt missing %q: %s", part, final)
		}
	}
	if strings.Contains(final, "Напиши короткий вывод") || strings.Contains(final, "•") {
		t.Fatalf("final reflection prompt kept old wording or bullets: %s", final)
	}
}

func TestRoutinePromptUsesDashExampleOnly(t *testing.T) {
	prompt := RoutinePlanPrompt()
	for _, part := range []string{"- зарядка", "- работа", "- английский перед сном", "- йога"} {
		if !strings.Contains(prompt, part) {
			t.Fatalf("routine prompt missing dash example %q: %s", part, prompt)
		}
	}
	if !strings.Contains(prompt, "Нумерацию тоже пойму") {
		t.Fatalf("routine prompt should mention numbered input support: %s", prompt)
	}
	for _, forbidden := range []string{"Можно использовать маркеры", "Максимум 9"} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("routine prompt should not mention %q: %s", forbidden, prompt)
		}
	}
}

func TestRoutineAlertsUseShortTrackmateStyle(t *testing.T) {
	checkin := postgres.RoutineCheckin{CheckinDate: time.Date(2026, 6, 28, 0, 0, 0, 0, time.UTC)}

	reminder := RoutineReminderText(checkin, "Игорь", "igor", 42)
	for _, part := range []string{"🔔 <b>Рутина за 28.06</b>", `<a href="tg://user?id=42">Игорь</a>`, "\n\nОтметь до полуночи"} {
		if !strings.Contains(reminder, part) {
			t.Fatalf("routine reminder missing %q: %s", part, reminder)
		}
	}
	if strings.Contains(reminder, "будут засчитаны") || strings.Contains(reminder, "еще не закрыта") || strings.Contains(reminder, "12:00") {
		t.Fatalf("routine reminder kept old wording: %s", reminder)
	}

	autoClosed := RoutineAutoClosedText(checkin, "Игорь", "igor", 42)
	for _, part := range []string{"⚠️ <b>Рутина за 28.06 закрыта</b>", "\n\nНеотмеченные пункты стали невыполненными"} {
		if !strings.Contains(autoClosed, part) {
			t.Fatalf("routine auto-close notice missing %q: %s", part, autoClosed)
		}
	}
}

func TestPersonalRoutineAlertCopyForEgor(t *testing.T) {
	reminderCheckin := postgres.RoutineCheckin{
		ID:          1,
		CheckinDate: time.Date(2026, 6, 28, 0, 0, 0, 0, time.UTC),
	}
	reminder := RoutineReminderText(reminderCheckin, "Егор Ковалец", "whysoxxx", 77)
	for _, part := range []string{`<a href="tg://user?id=77">Егор Ковалец</a>`, "\n\nЕгор, где рутина, бро?", "не будь нищим"} {
		if !strings.Contains(reminder, part) {
			t.Fatalf("personal routine reminder missing %q: %s", part, reminder)
		}
	}

	autoClosedCheckin := postgres.RoutineCheckin{
		ID:          3,
		CheckinDate: time.Date(2026, 6, 28, 0, 0, 0, 0, time.UTC),
	}
	autoClosed := RoutineAutoClosedText(autoClosedCheckin, "Егор Ковалец", "whysoxxx", 77)
	for _, part := range []string{`<a href="tg://user?id=77">Егор Ковалец</a>`, "\n\nЕгор, рутина ушла в минус", "Не будь нищим"} {
		if !strings.Contains(autoClosed, part) {
			t.Fatalf("personal routine auto-close missing %q: %s", part, autoClosed)
		}
	}
}

func TestPersonalDailyAlertCopyForEgor(t *testing.T) {
	alert := AlertText(domain.AlertDayClosedPendingReport, "Егор Ковалец", "whysoxxx", 77, "daily-alert:3:day_closed_pending_report")
	for _, part := range []string{`<a href="tg://user?id=77">Егор Ковалец</a>`, "\n\nЕгор, где дела, бро?", "Запиши результат, не будь нищим"} {
		if !strings.Contains(alert, part) {
			t.Fatalf("personal daily alert missing %q: %s", part, alert)
		}
	}

	defaultAlert := AlertText(domain.AlertDayClosedPendingReport, "Игорь", "igor", 42, "daily-alert:3:day_closed_pending_report")
	if strings.Contains(defaultAlert, "Егор") || !strings.Contains(defaultAlert, "День закончился") {
		t.Fatalf("default daily alert changed unexpectedly: %s", defaultAlert)
	}
}
