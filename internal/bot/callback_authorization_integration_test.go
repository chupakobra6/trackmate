package bot_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/igor/trackmate/internal/bot"
	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/logging"
	"github.com/igor/trackmate/internal/telegram"
	"github.com/igor/trackmate/internal/testsupport"
)

func TestOwnerlessDismissIsRejectedWithoutDeletingMessage(t *testing.T) {
	fake := newFakeTelegram()
	service := bot.NewService(nil, fake, logging.New("ERROR"), "UTC", 99)
	answer, err := service.HandleUpdate(context.Background(), telegram.Update{Callback: &telegram.CallbackQuery{
		ID:   "ownerless-dismiss",
		From: telegram.User{ID: 42, Username: "igor", FirstName: "Игорь"},
		Data: "notice:dismiss",
		Message: &telegram.Message{
			MessageID:       1001,
			MessageThreadID: 13,
			Chat:            telegram.Chat{ID: -1001234567000, Type: "supergroup", Title: "Group", IsForum: true},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(answer.Text, "больше не работает") {
		t.Fatalf("callback answer = %q", answer.Text)
	}
	if len(fake.deleted) != 0 || len(fake.edits) != 0 || len(fake.sent) != 0 || len(fake.replyMarkupEdits) != 0 {
		t.Fatalf("ownerless dismiss mutated Telegram: sent=%+v edits=%+v deleted=%+v markup=%+v", fake.sent, fake.edits, fake.deleted, fake.replyMarkupEdits)
	}
}

func TestPersonalCallbacksRejectOtherParticipantsWithoutMutation(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	ctx := context.Background()
	q := store.Queries()
	workspace, err := q.GetOrCreateWorkspace(ctx, -1001234567100, "Group", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	owner, err := q.RegisterParticipant(ctx, workspace.ID, 42, "igor", "Игорь")
	if err != nil {
		t.Fatal(err)
	}
	task, created, err := q.CreateDailyTask(ctx, workspace.ID, owner.ID, owner.UserID, time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC), "План дня", 201, 10)
	if err != nil || !created {
		t.Fatalf("task created=%v err=%v", created, err)
	}
	alert := createSentAlert(t, ctx, store, task.ID, 701)
	plan, err := q.UpsertRoutinePlan(ctx, workspace.ID, owner.ID, owner.UserID, []string{"зарядка"}, 301, 13)
	if err != nil {
		t.Fatal(err)
	}
	checkin, err := q.GetOrCreateRoutineCheckin(ctx, plan, time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	period, err := domain.CurrentGoalPeriod(workspace.Timezone, time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	goalSet, err := q.UpsertSeasonalGoalSet(ctx, workspace.ID, owner.ID, owner.UserID, period, "1. Работа", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name     string
		data     string
		message  int64
		threadID int64
	}{
		{name: "task report", data: fmt.Sprintf("task:report:%d", task.ID), message: 501, threadID: 10},
		{name: "task status", data: fmt.Sprintf("task:status:%d:done", task.ID), message: 502, threadID: 10},
		{name: "alert acknowledgement", data: fmt.Sprintf("alert:ack:%d", alert.ID), message: 701, threadID: 10},
		{name: "routine item", data: fmt.Sprintf("routine:item:%d:0:done", checkin.ID), message: 801, threadID: 13},
		{name: "goal final status", data: fmt.Sprintf("goals:final:%d:done", goalSet.ID), message: 901, threadID: 14},
		{name: "notice dismissal", data: "notice:dismiss:42", message: 1001, threadID: 13},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fake := newFakeTelegram()
			service := bot.NewService(store, fake, logging.New("ERROR"), "UTC", 99)
			answer, err := service.HandleUpdate(ctx, telegram.Update{Callback: &telegram.CallbackQuery{
				ID:   testCase.name,
				From: telegram.User{ID: 43, Username: "other", FirstName: "Другой"},
				Data: testCase.data,
				Message: &telegram.Message{
					MessageID:       testCase.message,
					MessageThreadID: testCase.threadID,
					Chat:            telegram.Chat{ID: workspace.ChatID, Type: "supergroup", Title: "Group", IsForum: true},
				},
			}})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(answer.Text, "только адресат") {
				t.Fatalf("callback answer = %q", answer.Text)
			}
			if len(fake.deleted) != 0 || len(fake.edits) != 0 || len(fake.sent) != 0 || len(fake.replyMarkupEdits) != 0 {
				t.Fatalf("foreign callback mutated Telegram: sent=%+v edits=%+v deleted=%+v markup=%+v", fake.sent, fake.edits, fake.deleted, fake.replyMarkupEdits)
			}
		})
	}

	storedTask, found, err := q.GetTask(ctx, task.ID)
	if err != nil || !found || !storedTask.Status.IsOpen() {
		t.Fatalf("foreign callbacks changed task: found=%v task=%+v err=%v", found, storedTask, err)
	}
	storedAlert, found, err := q.GetAlert(ctx, alert.ID)
	if err != nil || !found || storedAlert.AcknowledgedAt != nil {
		t.Fatalf("foreign callbacks changed alert: found=%v alert=%+v err=%v", found, storedAlert, err)
	}
	storedCheckin, found, err := q.GetRoutineCheckin(ctx, checkin.ID)
	if err != nil || !found || storedCheckin.Items[0].Status != nil {
		t.Fatalf("foreign callbacks changed routine: found=%v checkin=%+v err=%v", found, storedCheckin, err)
	}
	var finalReviewCount int
	if err := store.Pool().QueryRow(ctx, `SELECT count(*) FROM seasonal_goal_final_reviews WHERE goal_set_id = $1`, goalSet.ID).Scan(&finalReviewCount); err != nil {
		t.Fatal(err)
	}
	if finalReviewCount != 0 {
		t.Fatalf("foreign callback created %d goal final reviews", finalReviewCount)
	}
}

func TestOwnedCallbackRejectsResourceFromAnotherWorkspace(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	ctx := context.Background()
	task := createAlertLifecycleTask(t, ctx, store, -1001234567200)
	otherWorkspace, err := store.Queries().GetOrCreateWorkspace(ctx, -1001234567201, "Other Group", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	fake := newFakeTelegram()
	service := bot.NewService(store, fake, logging.New("ERROR"), "UTC", 99)

	answer, err := service.HandleUpdate(ctx, telegram.Update{Callback: &telegram.CallbackQuery{
		ID:   "cross-workspace-task",
		From: telegram.User{ID: task.OwnerUserID, Username: "igor", FirstName: "Игорь"},
		Data: fmt.Sprintf("task:report:%d", task.ID),
		Message: &telegram.Message{
			MessageID:       601,
			MessageThreadID: 10,
			Chat:            telegram.Chat{ID: otherWorkspace.ChatID, Type: "supergroup", Title: "Other Group", IsForum: true},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(answer.Text, "больше не работает") {
		t.Fatalf("callback answer = %q", answer.Text)
	}
	if len(fake.deleted) != 0 || len(fake.edits) != 0 || len(fake.sent) != 0 || len(fake.replyMarkupEdits) != 0 {
		t.Fatalf("cross-workspace callback mutated Telegram: sent=%+v edits=%+v deleted=%+v markup=%+v", fake.sent, fake.edits, fake.deleted, fake.replyMarkupEdits)
	}
}
