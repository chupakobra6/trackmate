package bot_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/igor/trackmate/internal/bot"
	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/logging"
	"github.com/igor/trackmate/internal/telegram"
	"github.com/igor/trackmate/internal/testsupport"
)

func TestRepeatedTaskStatusPreservesCardAndAcceptsReport(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
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
	task, created, err := q.CreateDailyTask(ctx, workspace.ID, participant.ID, participant.UserID, time.Now().UTC(), "Проверить повторное нажатие", 200, 10)
	if err != nil || !created {
		t.Fatalf("created=%v err=%v", created, err)
	}
	if err := q.SetDailyTaskCardMessageID(ctx, task.ID, 100); err != nil {
		t.Fatal(err)
	}
	fake := newFakeTelegram()
	service := bot.NewService(store, fake, logging.New("ERROR"), "UTC", 99)
	user := telegram.User{ID: participant.UserID, Username: "igor", FirstName: "Игорь"}
	card := telegram.Message{MessageID: 100, MessageThreadID: 10, Chat: telegram.Chat{ID: workspace.ChatID, Type: "supergroup", IsForum: true}}
	previousTask, created, err := q.CreateDailyTask(ctx, workspace.ID, participant.ID, participant.UserID, time.Now().UTC().AddDate(0, 0, -1), "Вчерашняя задача без отчёта", 199, 10)
	if err != nil || !created {
		t.Fatalf("previous task created=%v err=%v", created, err)
	}
	if err := q.SetDailyTaskCardMessageID(ctx, previousTask.ID, 99); err != nil {
		t.Fatal(err)
	}
	if err := q.UpdateTaskFailed(ctx, previousTask.ID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	previousCard := card
	previousCard.MessageID = 99
	if answer, err := service.HandleUpdate(ctx, telegram.Update{Callback: &telegram.CallbackQuery{ID: "previous-status", From: user, Data: fmt.Sprintf("task:status:%d:failed", previousTask.ID), Message: &previousCard}}); err != nil || answer.Text != "" {
		t.Fatalf("previous status answer=%+v err=%v", answer, err)
	}
	for i, status := range []domain.DailyTaskStatus{domain.DailyTaskDone, domain.DailyTaskDone, domain.DailyTaskPartial} {
		answer, err := service.HandleUpdate(ctx, telegram.Update{Callback: &telegram.CallbackQuery{ID: fmt.Sprint(i), From: user, Data: fmt.Sprintf("task:status:%d:%s", task.ID, status), Message: &card}})
		if err != nil || answer.Text != "" {
			t.Fatalf("status %s answer=%+v err=%v", status, answer, err)
		}
		if fake.wasDeleted(card.MessageID) {
			t.Fatalf("status replay deleted persistent card: %+v", fake.deleted)
		}
		if fake.wasDeleted(previousCard.MessageID) {
			t.Fatalf("switching reports deleted previous task card: %+v", fake.deleted)
		}
	}
	pending, found, err := q.GetPendingInput(ctx, workspace.ID, user.ID, 10)
	if err != nil || !found || pending.Payload["status"] != string(domain.DailyTaskPartial) {
		t.Fatalf("pending=%+v found=%v err=%v", pending, found, err)
	}
	report := telegram.Message{MessageID: 301, MessageThreadID: 10, DateUnix: time.Now().Unix(), From: &user, Chat: card.Chat, Text: "Сделал часть задачи"}
	if _, err := service.HandleUpdate(ctx, telegram.Update{Message: &report}); err != nil {
		t.Fatal(err)
	}
	updated, found, err := q.GetTask(ctx, task.ID)
	if err != nil || !found || updated.Status != domain.DailyTaskPartial || updated.ReportText == nil || *updated.ReportText != report.Text {
		t.Fatalf("task=%+v found=%v err=%v", updated, found, err)
	}
	if fake.wasDeleted(card.MessageID) {
		t.Fatalf("report deleted persistent card: %+v", fake.deleted)
	}
	if _, found, err := q.GetPendingInput(ctx, workspace.ID, user.ID, 10); err != nil || found {
		t.Fatalf("pending after report found=%v err=%v", found, err)
	}
}
