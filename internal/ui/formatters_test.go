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
			"task_html":    "Написать движок на Go",
			"report_html":  "Готово",
			"report_link":  "https://t.me/c/1/301?thread=10",
		},
	}
	got := FormatProgressEvent(event)
	for _, part := range []string{`<a href="tg://user?id=42">Igor</a>`, `<a href="https://t.me/c/1/301?thread=10">выполнил</a> задачу дня`, "Написать движок на Go", `<a href="https://t.me/c/1/301?thread=10">Готово</a>`} {
		if !strings.Contains(got, part) {
			t.Fatalf("formatted task event missing %q: %s", part, got)
		}
	}
	if strings.Contains(got, "<b>План:</b>\n\n<blockquote>") || strings.Contains(got, "<b>Итог:</b>\n\n<blockquote>") {
		t.Fatalf("formatted task event has extra blank line around blockquote: %s", got)
	}
}

func TestFormatDailyTaskCardShowsPlanWithoutExtraBlockquoteGap(t *testing.T) {
	task := postgres.DailyTask{
		Text:   "Подготовить короткий итог по задаче",
		Status: domain.DailyTaskActive,
	}
	got := FormatDailyTaskCard(task, "Игорь", "igor", "")
	for _, part := range []string{"Задача дня", "<b>План:</b>", "<b>Состояние:</b> в процессе"} {
		if !strings.Contains(got, part) {
			t.Fatalf("daily task card missing %q: %s", part, got)
		}
	}
	if strings.Contains(got, "<b>План:</b>\n\n<blockquote>") {
		t.Fatalf("daily task card has extra blank line before plan blockquote: %s", got)
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
	weekly := FormatGoalWeeklyReviewPrompt(goalSet, "Игорь", "igor")
	for _, part := range []string{
		"Что продвинулось по сезонным целям за эту неделю?",
		"Что мешало двигаться?",
		"Какой главный шаг берешь на следующую неделю?",
	} {
		if !strings.Contains(weekly, part) {
			t.Fatalf("weekly prompt missing %q: %s", part, weekly)
		}
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
	for _, part := range []string{"🔔 <b>Рутина за 28.06</b>", `<a href="tg://user?id=42">Игорь</a>`, "Отметь до полуночи"} {
		if !strings.Contains(reminder, part) {
			t.Fatalf("routine reminder missing %q: %s", part, reminder)
		}
	}
	if strings.Contains(reminder, "будут засчитаны") || strings.Contains(reminder, "еще не закрыта") || strings.Contains(reminder, "12:00") {
		t.Fatalf("routine reminder kept old wording: %s", reminder)
	}

	autoClosed := RoutineAutoClosedText(checkin, "Игорь", "igor", 42)
	for _, part := range []string{"⚠️ <b>Рутина за 28.06 закрыта</b>", "Неотмеченные пункты стали невыполненными"} {
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
	for _, part := range []string{`<a href="tg://user?id=77">Егор Ковалец</a>`, "Егор, где рутина, бро?", "не будь нищим по дисциплине"} {
		if !strings.Contains(reminder, part) {
			t.Fatalf("personal routine reminder missing %q: %s", part, reminder)
		}
	}

	autoClosedCheckin := postgres.RoutineCheckin{
		ID:          3,
		CheckinDate: time.Date(2026, 6, 28, 0, 0, 0, 0, time.UTC),
	}
	autoClosed := RoutineAutoClosedText(autoClosedCheckin, "Егор Ковалец", "whysoxxx", 77)
	for _, part := range []string{`<a href="tg://user?id=77">Егор Ковалец</a>`, "Егор, рутина ушла в минус", "Не будь нищим по дисциплине"} {
		if !strings.Contains(autoClosed, part) {
			t.Fatalf("personal routine auto-close missing %q: %s", part, autoClosed)
		}
	}
}

func TestPersonalDailyAlertCopyForEgor(t *testing.T) {
	alert := AlertText(domain.AlertDayClosedPendingReport, "Егор Ковалец", "whysoxxx", 77, "daily-alert:3:day_closed_pending_report")
	for _, part := range []string{`<a href="tg://user?id=77">Егор Ковалец</a>`, "Егор, где дела, бро?", "Закрой итог"} {
		if !strings.Contains(alert, part) {
			t.Fatalf("personal daily alert missing %q: %s", part, alert)
		}
	}

	defaultAlert := AlertText(domain.AlertDayClosedPendingReport, "Игорь", "igor", 42, "daily-alert:3:day_closed_pending_report")
	if strings.Contains(defaultAlert, "Егор") || !strings.Contains(defaultAlert, "День закончился") {
		t.Fatalf("default daily alert changed unexpectedly: %s", defaultAlert)
	}
}
