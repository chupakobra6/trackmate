package bot_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/igor/trackmate/internal/bot"
	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/logging"
	"github.com/igor/trackmate/internal/storage/postgres"
	"github.com/igor/trackmate/internal/telegram"
	"github.com/igor/trackmate/internal/testsupport"
)

func TestTaskReportRepeatedCallbackEditsOneMessageWithoutStackingPrompts(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	ctx := context.Background()
	task := createAlertLifecycleTask(t, ctx, store, -1001234567001)
	fake := newFakeTelegram()
	service := bot.NewService(store, fake, logging.New("ERROR"), "UTC", 99)
	callback := telegram.CallbackQuery{
		ID:   "task-report",
		From: telegram.User{ID: task.OwnerUserID, Username: "igor", FirstName: "Игорь"},
		Data: fmt.Sprintf("task:report:%d", task.ID),
		Message: &telegram.Message{
			MessageID:       200,
			MessageThreadID: 10,
			Chat:            telegram.Chat{ID: -1001234567001, Type: "supergroup", Title: "Group", IsForum: true},
		},
	}

	for i := 0; i < 3; i++ {
		callback.ID = fmt.Sprintf("task-report-%d", i)
		answer, err := service.HandleUpdate(ctx, telegram.Update{Callback: &callback})
		if err != nil || answer.Text != "" {
			t.Fatalf("callback %d answer=%+v err=%v", i, answer, err)
		}
	}

	if len(fake.sent) != 0 {
		t.Fatalf("report callbacks must not send stackable prompts: %+v", fake.sent)
	}
	if len(fake.edits) != 3 {
		t.Fatalf("edits=%d, want one in-place transition per callback", len(fake.edits))
	}
	for _, edit := range fake.edits {
		if edit.MessageID != 200 || !strings.Contains(edit.Text, "Выбери итог дня") || edit.ReplyMarkup == nil {
			t.Fatalf("unexpected in-place status prompt: %+v", edit)
		}
	}
	if _, found, err := store.Queries().GetPendingInput(ctx, task.WorkspaceGroupID, task.OwnerUserID, 10); err != nil || found {
		t.Fatalf("status selection must not create report pending yet: found=%v err=%v", found, err)
	}
}

func TestAlertAckUsesCallbackMessageToHealPreviouslyClearedMessageID(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	ctx := context.Background()
	task := createAlertLifecycleTask(t, ctx, store, -1001234567002)
	alert := createSentAlert(t, ctx, store, task.ID, 777)
	if err := store.Queries().AcknowledgeAlert(ctx, alert.ID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	fake := newFakeTelegram()
	service := bot.NewService(store, fake, logging.New("ERROR"), "UTC", 99)
	answer, err := service.HandleUpdate(ctx, alertAckUpdate(task, alert.ID, 777, -1001234567002))
	if err != nil || answer.Text == "" {
		t.Fatalf("answer=%+v err=%v", answer, err)
	}
	if !fake.wasDeleted(777) {
		t.Fatalf("legacy sticky alert was not closed through callback message id: %+v", fake.deleted)
	}
}

func TestAlertAckFallsBackToClosedTombstoneBeforeAcknowledging(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	ctx := context.Background()
	task := createAlertLifecycleTask(t, ctx, store, -1001234567003)
	alert := createSentAlert(t, ctx, store, task.ID, 777)
	fake := newFakeTelegram()
	fake.deleteErrors = map[int64]error{777: errors.New("Bad Request: message can't be deleted")}
	service := bot.NewService(store, fake, logging.New("ERROR"), "UTC", 99)

	if _, err := service.HandleUpdate(ctx, alertAckUpdate(task, alert.ID, 777, -1001234567003)); err != nil {
		t.Fatal(err)
	}
	edit, found := fake.findEdit(777)
	if !found || !strings.Contains(edit.Text, "Уведомление закрыто") || edit.ReplyMarkup == nil || len(edit.ReplyMarkup.InlineKeyboard) != 0 {
		t.Fatalf("alert fallback did not leave an inert tombstone: found=%v edit=%+v", found, edit)
	}
	updated, found, err := store.Queries().GetAlert(ctx, alert.ID)
	if err != nil || !found || updated.AcknowledgedAt == nil || updated.TelegramMessageID != nil {
		t.Fatalf("alert was not acknowledged after UI transition: found=%v alert=%+v err=%v", found, updated, err)
	}
}

func TestAlertAckKeepsDatabaseRetryableWhenAllTelegramTransitionsFail(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	ctx := context.Background()
	task := createAlertLifecycleTask(t, ctx, store, -1001234567004)
	alert := createSentAlert(t, ctx, store, task.ID, 777)
	fake := newFakeTelegram()
	fake.deleteErrors = map[int64]error{777: errors.New("delete failed")}
	fake.editErrors = map[int64]error{777: errors.New("edit failed")}
	fake.replyMarkupErrors = map[int64]error{777: errors.New("markup failed")}
	service := bot.NewService(store, fake, logging.New("ERROR"), "UTC", 99)

	if _, err := service.HandleUpdate(ctx, alertAckUpdate(task, alert.ID, 777, -1001234567004)); err == nil {
		t.Fatal("expected alert acknowledgement to fail while the message is still interactive")
	}
	updated, found, err := store.Queries().GetAlert(ctx, alert.ID)
	if err != nil || !found || updated.AcknowledgedAt != nil || updated.TelegramMessageID == nil || *updated.TelegramMessageID != 777 {
		t.Fatalf("failed UI transition must stay retryable: found=%v alert=%+v err=%v", found, updated, err)
	}
}

func createAlertLifecycleTask(t *testing.T, ctx context.Context, store *postgres.Store, chatID int64) postgres.DailyTask {
	t.Helper()
	q := store.Queries()
	workspace, err := q.GetOrCreateWorkspace(ctx, chatID, "Group", "UTC")
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
	task, created, err := q.CreateDailyTask(ctx, workspace.ID, participant.ID, participant.UserID, time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC), "План дня", 201, 10)
	if err != nil || !created {
		t.Fatalf("task created=%v err=%v", created, err)
	}
	if err := q.SetDailyTaskCardMessageID(ctx, task.ID, 200); err != nil {
		t.Fatal(err)
	}
	task, found, err := q.GetTask(ctx, task.ID)
	if err != nil || !found {
		t.Fatalf("task found=%v err=%v", found, err)
	}
	return task
}

func createSentAlert(t *testing.T, ctx context.Context, store *postgres.Store, taskID int64, messageID int64) postgres.DailyTaskAlert {
	t.Helper()
	alert, err := store.Queries().GetOrCreateAlert(ctx, taskID, domain.AlertDayClosedPendingReport)
	if err != nil {
		t.Fatal(err)
	}
	claimed, found, err := store.Queries().ClaimPendingAlert(ctx)
	if err != nil || !found || claimed.ID != alert.ID {
		t.Fatalf("alert claim=%+v found=%v err=%v", claimed, found, err)
	}
	if err := store.Queries().MarkAlertSent(ctx, claimed.ID, messageID); err != nil {
		t.Fatal(err)
	}
	alert, found, err = store.Queries().GetAlert(ctx, alert.ID)
	if err != nil || !found {
		t.Fatalf("alert found=%v err=%v", found, err)
	}
	return alert
}

func alertAckUpdate(task postgres.DailyTask, alertID int64, messageID int64, chatID int64) telegram.Update {
	return telegram.Update{Callback: &telegram.CallbackQuery{
		ID:   "alert-ack",
		From: telegram.User{ID: task.OwnerUserID, Username: "igor", FirstName: "Игорь"},
		Data: fmt.Sprintf("alert:ack:%d", alertID),
		Message: &telegram.Message{
			MessageID:       messageID,
			MessageThreadID: 10,
			Chat:            telegram.Chat{ID: chatID, Type: "supergroup", Title: "Group", IsForum: true},
		},
	}}
}
