package ui

import (
	"strings"
	"testing"

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
		},
	}
	got := FormatProgressEvent(event)
	for _, part := range []string{"выполнил задачу дня", "Написать движок на Go", "Готово"} {
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

func TestFinalCopyUsesCalmStyleAndDashLists(t *testing.T) {
	if !strings.Contains(TodayControlText, "Здесь у каждого одна главная задача дня") {
		t.Fatalf("today control text should keep old topic style: %s", TodayControlText)
	}
	if !strings.Contains(RoutineControlText, "Здесь живут повторяющиеся действия") {
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
		"Какие шаги сделаны по сезонным целям за эту неделю?",
		"С какими сложностями пришлось столкнуться?",
		"Какой главный шаг планируешь на следующую неделю?",
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
