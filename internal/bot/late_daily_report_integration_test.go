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

func TestAutoFailedTaskAcceptsOneLateReportWithoutChangingFailedStatus(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	ctx := context.Background()
	q := store.Queries()
	workspace, err := q.GetOrCreateWorkspace(ctx, -1001234567010, "Group", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertTopicBinding(ctx, workspace.ID, domain.TopicToday, 10, "Сегодня"); err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertTopicBinding(ctx, workspace.ID, domain.TopicProgress, 11, "Прогресс"); err != nil {
		t.Fatal(err)
	}
	participant, err := q.RegisterParticipant(ctx, workspace.ID, 42, "igor", "Игорь")
	if err != nil {
		t.Fatal(err)
	}
	task, created, err := q.CreateDailyTask(ctx, workspace.ID, participant.ID, participant.UserID, time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC), "План дня", 201, 10)
	if err != nil || !created {
		t.Fatalf("task created=%v err=%v", created, err)
	}
	if err := q.SetDailyTaskCardMessageID(ctx, task.ID, 200); err != nil {
		t.Fatal(err)
	}
	autoFailedAt := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	if err := q.UpdateTaskFailed(ctx, task.ID, autoFailedAt); err != nil {
		t.Fatal(err)
	}
	failedTask, found, err := q.GetTask(ctx, task.ID)
	if err != nil || !found {
		t.Fatalf("failed task found=%v err=%v", found, err)
	}
	if err := q.CreateAutoFailProgressEvent(ctx, failedTask, workspace, participant, 10); err != nil {
		t.Fatal(err)
	}
	progress, ok, err := q.ClaimProgressEvent(ctx)
	if err != nil || !ok {
		t.Fatalf("progress claim ok=%v err=%v", ok, err)
	}
	if err := q.MarkProgressEventPublished(ctx, progress.ID, 500, autoFailedAt); err != nil {
		t.Fatal(err)
	}
	alert, err := q.GetOrCreateAlert(ctx, task.ID, domain.AlertOverdueTaskFailed)
	if err != nil {
		t.Fatal(err)
	}
	claimedAlert, ok, err := q.ClaimPendingAlert(ctx)
	if err != nil || !ok || claimedAlert.ID != alert.ID {
		t.Fatalf("alert claim=%+v ok=%v err=%v", claimedAlert, ok, err)
	}
	if err := q.MarkAlertSent(ctx, alert.ID, 777); err != nil {
		t.Fatal(err)
	}

	fake := newFakeTelegram()
	service := bot.NewService(store, fake, logging.New("ERROR"), "UTC", 99)
	callbackMessage := &telegram.Message{
		MessageID:       777,
		MessageThreadID: 10,
		Chat:            telegram.Chat{ID: workspace.ChatID, Type: "supergroup", Title: "Group", IsForum: true},
	}
	user := telegram.User{ID: participant.UserID, Username: "igor", FirstName: "Игорь"}
	answer, err := service.HandleUpdate(ctx, telegram.Update{Callback: &telegram.CallbackQuery{
		ID: "late-report", From: user, Data: fmt.Sprintf("task:report:%d", task.ID), Message: callbackMessage,
	}})
	if err != nil || answer.Text != "" {
		t.Fatalf("late report callback answer=%+v err=%v", answer, err)
	}
	statusEdit, found := fake.findEdit(777)
	if !found || !strings.Contains(statusEdit.Text, "Выбери итог дня") || statusEdit.ReplyMarkup == nil {
		t.Fatalf("late report status chooser mismatch: found=%v edit=%+v", found, statusEdit)
	}

	answer, err = service.HandleUpdate(ctx, telegram.Update{Callback: &telegram.CallbackQuery{
		ID: "late-status", From: user, Data: fmt.Sprintf("task:status:%d:done", task.ID), Message: callbackMessage,
	}})
	if err != nil || answer.Text != "" {
		t.Fatalf("late status callback answer=%+v err=%v", answer, err)
	}
	promptEdit, found := fake.findEdit(777)
	if !found || !strings.Contains(promptEdit.Text, "Напиши результат одним сообщением") {
		t.Fatalf("late report prompt mismatch: found=%v edit=%+v", found, promptEdit)
	}

	report := telegram.Message{
		MessageID:       301,
		MessageThreadID: 10,
		DateUnix:        autoFailedAt.Add(5 * time.Minute).Unix(),
		From:            &user,
		Chat:            callbackMessage.Chat,
		Text:            "Фактический результат после дедлайна",
	}
	if _, err := service.HandleUpdate(ctx, telegram.Update{Message: &report}); err != nil {
		t.Fatal(err)
	}

	updated, found, err := q.GetTask(ctx, task.ID)
	if err != nil || !found {
		t.Fatalf("updated task found=%v err=%v", found, err)
	}
	if updated.Status != domain.DailyTaskFailed || updated.ReportStatus == nil || *updated.ReportStatus != domain.DailyTaskDone || updated.ReportText == nil || *updated.ReportText != report.Text || updated.ReportMessageID == nil || *updated.ReportMessageID != report.MessageID {
		t.Fatalf("unexpected late report state: %+v", updated)
	}
	cardEdit, found := fake.findEdit(200)
	if !found || !strings.Contains(cardEdit.Text, "не выполнил задачу дня") || !strings.Contains(cardEdit.Text, report.Text) || cardEdit.ReplyMarkup != nil {
		t.Fatalf("failed task card was not updated with late report: found=%v edit=%+v", found, cardEdit)
	}
	progressEdit, found := fake.findEdit(500)
	if !found || !strings.Contains(progressEdit.Text, "не выполнил") || !strings.Contains(progressEdit.Text, report.Text) {
		t.Fatalf("auto-fail progress was not updated with late report: found=%v edit=%+v", found, progressEdit)
	}
	if !fake.wasDeleted(777) {
		t.Fatalf("late-report alert prompt was not removed: %+v", fake.deleted)
	}

	var autoFailedEvents, closedEvents int
	if err := store.Pool().QueryRow(ctx, `
SELECT count(*) FILTER (WHERE event_type = 'daily_task.auto_failed'),
       count(*) FILTER (WHERE event_type = 'daily_task.closed')
FROM progress_events
WHERE daily_task_id = $1
`, task.ID).Scan(&autoFailedEvents, &closedEvents); err != nil {
		t.Fatal(err)
	}
	if autoFailedEvents != 1 || closedEvents != 0 {
		t.Fatalf("progress events auto_failed=%d closed=%d", autoFailedEvents, closedEvents)
	}

	answer, err = service.HandleUpdate(ctx, telegram.Update{Callback: &telegram.CallbackQuery{
		ID: "second-late-report", From: user, Data: fmt.Sprintf("task:report:%d", task.ID), Message: callbackMessage,
	}})
	if err != nil || !strings.Contains(answer.Text, "уже закрыта") {
		t.Fatalf("second late report answer=%+v err=%v", answer, err)
	}
}
