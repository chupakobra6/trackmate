package bot_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/igor/trackmate/internal/bot"
	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/logging"
	"github.com/igor/trackmate/internal/telegram"
	"github.com/igor/trackmate/internal/testsupport"
)

func TestDailyReportSaveDeletesPromptWithoutConfirmation(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	fake := newFakeTelegram()
	service := bot.NewService(store, fake, logging.New("ERROR"), "UTC", 99)
	ctx := context.Background()
	q := store.Queries()

	workspace, err := q.GetOrCreateWorkspace(ctx, -1001234567890, "Group", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertTopicBinding(ctx, workspace.ID, domain.TopicToday, 10, "Сегодня"); err != nil {
		t.Fatal(err)
	}
	participant, err := q.RegisterParticipant(ctx, workspace.ID, 42, "igor", "Игорь")
	if err != nil {
		t.Fatal(err)
	}
	task, created, err := q.CreateDailyTask(ctx, workspace.ID, participant.ID, participant.UserID, time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC), "Проверить результат", 200, 10)
	if err != nil || !created {
		t.Fatalf("task created=%v err=%v", created, err)
	}
	if err := q.SetDailyTaskCardMessageID(ctx, task.ID, 100); err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertPendingInput(ctx, workspace.ID, participant.UserID, 10, domain.PendingDailyTaskReport, map[string]any{
		"thread_id":         10,
		"task_id":           task.ID,
		"status":            string(domain.DailyTaskDone),
		"prompt_message_id": 101,
	}); err != nil {
		t.Fatal(err)
	}

	report := telegram.Message{
		MessageID:       201,
		MessageThreadID: 10,
		DateUnix:        time.Now().Unix(),
		From:            &telegram.User{ID: participant.UserID, Username: "igor", FirstName: "Игорь"},
		Chat:            telegram.Chat{ID: workspace.ChatID, Type: "supergroup", Title: "Group", IsForum: true},
		Text:            "Результат записан",
	}
	if _, err := service.HandleUpdate(ctx, telegram.Update{UpdateID: 1, Message: &report}); err != nil {
		t.Fatal(err)
	}

	if !fake.wasDeleted(101) {
		t.Fatalf("result prompt was not deleted after save: %+v", fake.deleted)
	}
	if fake.hasSentToThread(10, "Результат сохранен") {
		t.Fatalf("daily result save confirmation should not be sent: %+v", fake.sent)
	}
	edit, ok := fake.findEdit(100)
	if !ok || !containsAll(edit.Text, "✅ <b>Задача дня</b> @igor", "<b>Результат:</b>", "Результат записан") {
		t.Fatalf("daily card edit mismatch: found=%v edit=%+v", ok, edit)
	}
	if containsAll(edit.Text, "<b>Состояние:</b>") || containsAll(edit.Text, "<b>Статус:</b>") {
		t.Fatalf("daily card should not render a separate status line: %s", edit.Text)
	}
}

func containsAll(value string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(value, part) {
			return false
		}
	}
	return true
}
