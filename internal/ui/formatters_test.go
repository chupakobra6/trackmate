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
			"task_html":    "Написать Go runtime",
			"report_html":  "Готово",
		},
	}
	got := FormatProgressEvent(event)
	for _, part := range []string{"выполнил задачу дня", "Написать Go runtime", "Готово"} {
		if !strings.Contains(got, part) {
			t.Fatalf("formatted task event missing %q: %s", part, got)
		}
	}
}

func TestFormatProgressEventCustomUpdate(t *testing.T) {
	event := postgres.ProgressEvent{
		EventType: domain.ProgressCustomUpdate,
		Payload: map[string]any{
			"title": "Встречайте: Trackmate 1.0 на Go",
			"body":  "Бот переехал на новый runtime.",
			"items": []any{
				"сохранили задачи, отчеты и прогресс",
				"удалили старый Materials",
			},
		},
	}
	got := FormatProgressEvent(event)
	for _, part := range []string{"Встречайте: Trackmate 1.0 на Go", "новый runtime", "сохранили задачи", "удалили старый Materials"} {
		if !strings.Contains(got, part) {
			t.Fatalf("formatted custom update missing %q: %s", part, got)
		}
	}
}

func TestFormatRoutineLeaderboardShowsRateStreakAndItemCount(t *testing.T) {
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
	for _, part := range []string{"92% за 7 дней", "стрик 5 дней", "4 пункта", "Лучший стрик сезона"} {
		if !strings.Contains(got, part) {
			t.Fatalf("leaderboard missing %q: %s", part, got)
		}
	}
}
