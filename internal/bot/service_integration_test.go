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

func TestSetupCreatesTodayRoutineGoalsAndProgress(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	fake := newFakeTelegram()
	service := bot.NewService(store, fake, logging.New("ERROR"), "UTC", 99)
	answer, err := service.HandleUpdate(context.Background(), telegram.Update{Callback: &telegram.CallbackQuery{
		ID:   "cb-1",
		From: telegram.User{ID: 10, FirstName: "Admin"},
		Data: "setup:start",
		Message: &telegram.Message{
			MessageID: 1,
			Chat:      telegram.Chat{ID: -1001234567890, Type: "supergroup", Title: "Group", IsForum: true},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if answer.Text != "" {
		t.Fatalf("unexpected callback answer: %q", answer.Text)
	}
	if len(fake.createdTopics) != 4 {
		t.Fatalf("expected four active topics, got %v", fake.createdTopics)
	}
	if fake.createdTopics[0] != "Сегодня" || fake.createdTopics[1] != "Рутины" || fake.createdTopics[2] != "Цели" || fake.createdTopics[3] != "Прогресс" {
		t.Fatalf("unexpected topics: %v", fake.createdTopics)
	}
	workspace, found, err := store.Queries().GetWorkspaceByChatID(context.Background(), -1001234567890)
	if err != nil || !found {
		t.Fatalf("workspace found=%v err=%v", found, err)
	}
	bindings, err := store.Queries().ListTopicBindings(context.Background(), workspace.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := bindings[domain.TopicToday]; !ok {
		t.Fatal("today binding missing")
	}
	if _, ok := bindings[domain.TopicProgress]; !ok {
		t.Fatal("progress binding missing")
	}
	if _, ok := bindings[domain.TopicRoutine]; !ok {
		t.Fatal("routine binding missing")
	}
	if _, ok := bindings[domain.TopicGoals]; !ok {
		t.Fatal("goals binding missing")
	}
}

func TestPhotoAlbumReportConsumesPendingInputOnce(t *testing.T) {
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
	if _, err := q.UpsertTopicBinding(ctx, workspace.ID, domain.TopicProgress, 11, "Прогресс"); err != nil {
		t.Fatal(err)
	}
	participant, err := q.RegisterParticipant(ctx, workspace.ID, 42, "igor", "Игорь")
	if err != nil {
		t.Fatal(err)
	}
	task, created, err := q.CreateDailyTask(ctx, workspace.ID, participant.ID, participant.UserID, time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC), "Приложить фото к итогу", 200, 10)
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

	first := telegram.Message{
		MessageID:       201,
		MessageThreadID: 10,
		DateUnix:        time.Now().Unix(),
		From:            &telegram.User{ID: participant.UserID, Username: "igor", FirstName: "Игорь"},
		Chat:            telegram.Chat{ID: workspace.ChatID, Type: "supergroup", Title: "Group", IsForum: true},
		Caption:         "Фото к итогу: задача закрыта двумя изображениями.",
		MediaGroupID:    "album-1",
		Photo:           []telegram.PhotoSize{{}},
	}
	second := first
	second.MessageID = 202

	if _, err := service.HandleUpdate(ctx, telegram.Update{UpdateID: 1, Message: &first}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.HandleUpdate(ctx, telegram.Update{UpdateID: 2, Message: &second}); err != nil {
		t.Fatal(err)
	}

	updated, found, err := q.GetTask(ctx, task.ID)
	if err != nil || !found {
		t.Fatalf("task found=%v err=%v", found, err)
	}
	if updated.Status != domain.DailyTaskDone || updated.ReportText == nil || *updated.ReportText != "Фото к итогу: задача закрыта двумя изображениями." {
		t.Fatalf("unexpected report state: %+v", updated)
	}
	if _, found, err := q.GetPendingInput(ctx, workspace.ID, participant.UserID, 10); err != nil || found {
		t.Fatalf("pending input found=%v err=%v", found, err)
	}
	var progressCount int
	if err := store.Pool().QueryRow(ctx, `
SELECT count(*)
FROM progress_events
WHERE workspace_group_id = $1 AND daily_task_id = $2 AND event_type = 'daily_task.closed'
`, workspace.ID, task.ID).Scan(&progressCount); err != nil {
		t.Fatal(err)
	}
	if progressCount != 1 {
		t.Fatalf("progress events = %d, want 1", progressCount)
	}
}

func TestEditedTaskMessageUpdatesStoredTextAndCard(t *testing.T) {
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
	task, created, err := q.CreateDailyTask(ctx, workspace.ID, participant.ID, participant.UserID, time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC), "Старая задача", 201, 10)
	if err != nil || !created {
		t.Fatalf("task created=%v err=%v", created, err)
	}
	if err := q.SetDailyTaskCardMessageID(ctx, task.ID, 100); err != nil {
		t.Fatal(err)
	}

	edited := telegram.Message{
		MessageID:       201,
		MessageThreadID: 10,
		DateUnix:        time.Now().Unix(),
		From:            &telegram.User{ID: participant.UserID, Username: "igor", FirstName: "Игорь"},
		Chat:            telegram.Chat{ID: workspace.ChatID, Type: "supergroup", Title: "Group", IsForum: true},
		Text:            "Новая задача",
	}
	if _, err := service.HandleUpdate(ctx, telegram.Update{UpdateID: 3, EditedMessage: &edited}); err != nil {
		t.Fatal(err)
	}

	updated, found, err := q.GetTask(ctx, task.ID)
	if err != nil || !found {
		t.Fatalf("task found=%v err=%v", found, err)
	}
	if updated.Text != "Новая задача" {
		t.Fatalf("task text = %q", updated.Text)
	}
	edit, ok := fake.findEdit(100)
	if !ok || !strings.Contains(edit.Text, "Новая задача") {
		t.Fatalf("task card edit missing new text: found=%v edit=%+v", ok, edit)
	}
}

func TestEditedReportMessageUpdatesPublishedProgress(t *testing.T) {
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
	if _, err := q.UpsertTopicBinding(ctx, workspace.ID, domain.TopicProgress, 11, "Прогресс"); err != nil {
		t.Fatal(err)
	}
	participant, err := q.RegisterParticipant(ctx, workspace.ID, 42, "igor", "Игорь")
	if err != nil {
		t.Fatal(err)
	}
	task, created, err := q.CreateDailyTask(ctx, workspace.ID, participant.ID, participant.UserID, time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC), "План дня", 201, 10)
	if err != nil || !created {
		t.Fatalf("task created=%v err=%v", created, err)
	}
	if err := q.SetDailyTaskCardMessageID(ctx, task.ID, 100); err != nil {
		t.Fatal(err)
	}
	submitted, err := q.SubmitTaskReport(ctx, task.ID, participant.UserID, domain.DailyTaskDone, "Старый итог", "Игорь", 301, 10)
	if err != nil || !submitted {
		t.Fatalf("submitted=%v err=%v", submitted, err)
	}
	events, err := q.ListPendingProgressEvents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("progress events = %d, want 1", len(events))
	}
	if err := q.MarkProgressEventPublished(ctx, events[0].ID, 500, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	edited := telegram.Message{
		MessageID:       301,
		MessageThreadID: 10,
		DateUnix:        time.Now().Unix(),
		From:            &telegram.User{ID: participant.UserID, Username: "igor", FirstName: "Игорь"},
		Chat:            telegram.Chat{ID: workspace.ChatID, Type: "supergroup", Title: "Group", IsForum: true},
		Text:            "Новый итог",
	}
	if _, err := service.HandleUpdate(ctx, telegram.Update{UpdateID: 4, EditedMessage: &edited}); err != nil {
		t.Fatal(err)
	}

	updated, found, err := q.GetTask(ctx, task.ID)
	if err != nil || !found {
		t.Fatalf("task found=%v err=%v", found, err)
	}
	if updated.ReportText == nil || *updated.ReportText != "Новый итог" {
		t.Fatalf("report text = %v", updated.ReportText)
	}
	cardEdit, ok := fake.findEdit(100)
	if !ok || !strings.Contains(cardEdit.Text, "Новый итог") {
		t.Fatalf("task card edit missing new report: found=%v edit=%+v", ok, cardEdit)
	}
	progressEdit, ok := fake.findEdit(500)
	if !ok || !strings.Contains(progressEdit.Text, "Новый итог") || !strings.Contains(progressEdit.Text, `<a href="https://t.me/c/1234567890/301?thread=10">выполнил</a>`) || strings.Contains(progressEdit.Text, "Старый итог") {
		t.Fatalf("progress edit mismatch: found=%v edit=%+v", ok, progressEdit)
	}
	var reportHTML, reportLink string
	if err := store.Pool().QueryRow(ctx, `
SELECT payload ->> 'report_html', payload ->> 'report_link'
FROM progress_events
WHERE id = $1
`, events[0].ID).Scan(&reportHTML, &reportLink); err != nil {
		t.Fatal(err)
	}
	if reportHTML != "Новый итог" {
		t.Fatalf("progress payload report_html = %q", reportHTML)
	}
	if reportLink != "https://t.me/c/1234567890/301?thread=10" {
		t.Fatalf("progress payload report_link = %q", reportLink)
	}
}

func TestRoutinePlanSaveDeletesSetupMessagesWithoutConfirmation(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	fake := newFakeTelegram()
	service := bot.NewService(store, fake, logging.New("ERROR"), "UTC", 99)
	ctx := context.Background()
	q := store.Queries()

	workspace, err := q.GetOrCreateWorkspace(ctx, -1001234567890, "Group", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertTopicBinding(ctx, workspace.ID, domain.TopicRoutine, 13, "Рутины"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.HandleUpdate(ctx, telegram.Update{Callback: &telegram.CallbackQuery{
		ID:   "routine-configure",
		From: telegram.User{ID: 42, Username: "igor", FirstName: "Игорь"},
		Data: "routine:configure",
		Message: &telegram.Message{
			MessageID:       200,
			MessageThreadID: 13,
			Chat:            telegram.Chat{ID: workspace.ChatID, Type: "supergroup", Title: "Group", IsForum: true},
		},
	}}); err != nil {
		t.Fatal(err)
	}
	if len(fake.sent) != 1 || !strings.Contains(fake.sent[0].Text, "Пришли рутину") {
		t.Fatalf("expected routine setup prompt, got %+v", fake.sent)
	}

	input := telegram.Message{
		MessageID:       301,
		MessageThreadID: 13,
		DateUnix:        time.Now().Unix(),
		From:            &telegram.User{ID: 42, Username: "igor", FirstName: "Игорь"},
		Chat:            telegram.Chat{ID: workspace.ChatID, Type: "supergroup", Title: "Group", IsForum: true},
		Text:            "1. Зарядка\n5. Работа\n2) Сон до 12",
	}
	if _, err := service.HandleUpdate(ctx, telegram.Update{Message: &input}); err != nil {
		t.Fatal(err)
	}
	if !fake.wasDeleted(1001) || !fake.wasDeleted(301) {
		t.Fatalf("routine setup prompt and input should be deleted, deleted=%+v", fake.deleted)
	}
	if len(fake.edits) != 0 {
		t.Fatalf("routine save should not leave confirmation edits: %+v", fake.edits)
	}
	if len(fake.sent) != 1 {
		t.Fatalf("routine save should not send confirmation messages, sent=%+v", fake.sent)
	}
	participant, err := q.RegisterParticipant(ctx, workspace.ID, 42, "igor", "Игорь")
	if err != nil {
		t.Fatal(err)
	}
	plan, found, err := q.GetRoutinePlan(ctx, workspace.ID, participant.ID)
	if err != nil || !found {
		t.Fatalf("routine plan found=%v err=%v", found, err)
	}
	want := []string{"Зарядка", "Работа", "Сон до 12"}
	if len(plan.Items) != len(want) {
		t.Fatalf("routine items = %v want %v", plan.Items, want)
	}
	for i := range want {
		if plan.Items[i] != want[i] {
			t.Fatalf("routine items = %v want %v", plan.Items, want)
		}
	}
}

func TestRoutinePlanChangeSnapshotsPreviousRoutineBeforeSavingNewList(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	fake := newFakeTelegram()
	service := bot.NewService(store, fake, logging.New("ERROR"), "UTC", 99)
	ctx := context.Background()
	q := store.Queries()

	workspace, err := q.GetOrCreateWorkspace(ctx, -1001234567890, "Group", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	participant, err := q.RegisterParticipant(ctx, workspace.ID, 42, "igor", "Игорь")
	if err != nil {
		t.Fatal(err)
	}
	oldPlan, err := q.UpsertRoutinePlan(ctx, workspace.ID, participant.ID, participant.UserID, []string{"старая зарядка", "старый сон"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Pool().Exec(ctx, `UPDATE routine_plans SET created_at = $2 WHERE id = $1`, oldPlan.ID, time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 6, 24, 7, 30, 0, 0, time.UTC)
	if err := q.SetClockOverride(ctx, &now); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = q.SetClockOverride(ctx, nil) }()
	if _, err := q.UpsertPendingInput(ctx, workspace.ID, participant.UserID, 13, domain.PendingRoutinePlan, map[string]any{
		"thread_id":         13,
		"prompt_message_id": 100,
	}); err != nil {
		t.Fatal(err)
	}

	input := telegram.Message{
		MessageID:       301,
		MessageThreadID: 13,
		DateUnix:        now.Unix(),
		From:            &telegram.User{ID: participant.UserID, Username: "igor", FirstName: "Игорь"},
		Chat:            telegram.Chat{ID: workspace.ChatID, Type: "supergroup", Title: "Group", IsForum: true},
		Text:            "- новая работа\n- новый спорт",
	}
	if _, err := service.HandleUpdate(ctx, telegram.Update{Message: &input}); err != nil {
		t.Fatal(err)
	}

	yesterday, found, err := q.GetRoutineCheckinForDate(ctx, workspace.ID, participant.ID, time.Date(2026, 6, 23, 0, 0, 0, 0, time.UTC))
	if err != nil || !found {
		t.Fatalf("previous checkin found=%v err=%v", found, err)
	}
	if got := []string{yesterday.Items[0].Text, yesterday.Items[1].Text}; fmt.Sprint(got) != "[старая зарядка старый сон]" {
		t.Fatalf("previous checkin should keep old routine, got %v", got)
	}
	newPlan, found, err := q.GetRoutinePlan(ctx, workspace.ID, participant.ID)
	if err != nil || !found {
		t.Fatalf("new plan found=%v err=%v", found, err)
	}
	if fmt.Sprint(newPlan.Items) != "[новая работа новый спорт]" {
		t.Fatalf("new routine was not saved: %v", newPlan.Items)
	}
}

func TestRoutineCheckinFlowStaysInRoutineTopic(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	fake := newFakeTelegram()
	service := bot.NewService(store, fake, logging.New("ERROR"), "UTC", 99)
	ctx := context.Background()
	q := store.Queries()

	workspace, err := q.GetOrCreateWorkspace(ctx, -1001234567890, "Group", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertTopicBinding(ctx, workspace.ID, domain.TopicRoutine, 13, "Рутины"); err != nil {
		t.Fatal(err)
	}
	routineTableMessageID := int64(900)
	if err := q.SetTopicMessages(ctx, workspace.ID, domain.TopicRoutine, &routineTableMessageID, nil, false, false); err != nil {
		t.Fatal(err)
	}
	participant, err := q.RegisterParticipant(ctx, workspace.ID, 42, "igor", "Игорь")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := q.UpsertRoutinePlan(ctx, workspace.ID, participant.ID, participant.UserID, []string{"зарядка", "йога"})
	if err != nil {
		t.Fatal(err)
	}
	checkin, err := q.GetOrCreateRoutineCheckin(ctx, plan, time.Date(2026, 6, 24, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if err := q.SetRoutineCheckinCardMessageID(ctx, checkin.ID, 100, 13); err != nil {
		t.Fatal(err)
	}

	cardMessage := &telegram.Message{
		MessageID:       100,
		MessageThreadID: 13,
		Chat:            telegram.Chat{ID: workspace.ChatID, Type: "supergroup", Title: "Group", IsForum: true},
	}
	if _, err := service.HandleUpdate(ctx, telegram.Update{Callback: &telegram.CallbackQuery{
		ID:      "routine-done",
		From:    telegram.User{ID: participant.UserID, Username: "igor", FirstName: "Игорь"},
		Data:    fmt.Sprintf("routine:item:%d:0:done", checkin.ID),
		Message: cardMessage,
	}}); err != nil {
		t.Fatal(err)
	}
	edit, ok := fake.findEdit(100)
	if !ok || !strings.Contains(edit.Text, "✅ зарядка") || strings.Contains(edit.Text, "йога?") {
		t.Fatalf("expected routine status-only edit, found=%v edit=%+v", ok, edit)
	}

	if _, err := service.HandleUpdate(ctx, telegram.Update{Callback: &telegram.CallbackQuery{
		ID:      "routine-partial",
		From:    telegram.User{ID: participant.UserID, Username: "igor", FirstName: "Игорь"},
		Data:    fmt.Sprintf("routine:item:%d:1:partial", checkin.ID),
		Message: cardMessage,
	}}); err != nil {
		t.Fatal(err)
	}
	statusEdit, ok := fake.findEdit(100)
	if !ok || !strings.Contains(statusEdit.Text, "🔸 йога") || strings.Contains(statusEdit.Text, "Что помешало?") {
		t.Fatalf("expected main routine card to only mark status, found=%v edit=%+v", ok, statusEdit)
	}
	if len(fake.sent) != 1 {
		t.Fatalf("expected separate routine reason prompt, got %+v", fake.sent)
	}
	for _, part := range []string{"Что помешало?", "йога"} {
		if !strings.Contains(fake.sent[0].Text, part) {
			t.Fatalf("routine reason prompt missing %q: %s", part, fake.sent[0].Text)
		}
	}
	if fake.sent[0].ReplyToMessageID != 100 {
		t.Fatalf("routine reason prompt should reply to main card: %+v", fake.sent[0])
	}
	if pending, found, err := q.GetPendingInput(ctx, workspace.ID, participant.UserID, 13); err != nil || !found || pending.Kind != domain.PendingRoutineReason {
		t.Fatalf("routine reason pending found=%v pending=%+v err=%v", found, pending, err)
	}

	reason := telegram.Message{
		MessageID:       301,
		MessageThreadID: 13,
		DateUnix:        time.Now().Unix(),
		From:            &telegram.User{ID: participant.UserID, Username: "igor", FirstName: "Игорь"},
		Chat:            telegram.Chat{ID: workspace.ChatID, Type: "supergroup", Title: "Group", IsForum: true},
		Text:            "Сорвался график",
	}
	if _, err := service.HandleUpdate(ctx, telegram.Update{Message: &reason}); err != nil {
		t.Fatal(err)
	}
	if _, found, err := q.GetPendingInput(ctx, workspace.ID, participant.UserID, 13); err != nil || found {
		t.Fatalf("routine pending should be cleared found=%v err=%v", found, err)
	}
	updated, found, err := q.GetRoutineCheckin(ctx, checkin.ID)
	if err != nil || !found {
		t.Fatalf("routine checkin found=%v err=%v", found, err)
	}
	if updated.CompletedAt == nil || updated.ReflectionText != nil {
		t.Fatalf("routine not completed: %+v", updated)
	}
	if !fake.wasDeleted(1001) || !fake.wasDeleted(301) || !fake.wasDeleted(100) {
		t.Fatalf("routine reason prompt and answer should be deleted, deleted=%+v", fake.deleted)
	}
	tableEdit, ok := fake.findEdit(900)
	if !ok || !strings.Contains(tableEdit.Text, "Таблица рутин") {
		t.Fatalf("routine table edit missing: found=%v edit=%+v", ok, tableEdit)
	}
	var progressCount int
	if err := store.Pool().QueryRow(ctx, `SELECT count(*) FROM progress_events WHERE workspace_group_id = $1`, workspace.ID).Scan(&progressCount); err != nil {
		t.Fatal(err)
	}
	if progressCount != 0 {
		t.Fatalf("routine flow must not publish progress events, got %d", progressCount)
	}
}

func TestWrongTopicPendingInputIsIgnoredAndKept(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	fake := newFakeTelegram()
	service := bot.NewService(store, fake, logging.New("ERROR"), "UTC", 99)
	ctx := context.Background()
	q := store.Queries()

	workspace, err := q.GetOrCreateWorkspace(ctx, -1001234567890, "Group", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertPendingInput(ctx, workspace.ID, 42, 13, domain.PendingRoutinePlan, map[string]any{
		"thread_id":         13,
		"prompt_message_id": 100,
	}); err != nil {
		t.Fatal(err)
	}

	message := telegram.Message{
		MessageID:       301,
		MessageThreadID: 14,
		DateUnix:        time.Now().Unix(),
		From:            &telegram.User{ID: 42, Username: "igor", FirstName: "Игорь"},
		Chat:            telegram.Chat{ID: workspace.ChatID, Type: "supergroup", Title: "Group", IsForum: true},
		Text:            "1. Работа\n— Результат: проверить цели",
	}
	if _, err := service.HandleUpdate(ctx, telegram.Update{Message: &message}); err != nil {
		t.Fatal(err)
	}
	pending, found, err := q.GetPendingInput(ctx, workspace.ID, 42, 13)
	if err != nil || !found || pending.Kind != domain.PendingRoutinePlan {
		t.Fatalf("routine pending should remain found=%v pending=%+v err=%v", found, pending, err)
	}
	if len(fake.deleted) != 0 {
		t.Fatalf("wrong-topic input should not delete messages, got %+v", fake.deleted)
	}
}

func TestConfigureGoalsDoesNotTouchUnfinishedRoutineDraft(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	fake := newFakeTelegram()
	service := bot.NewService(store, fake, logging.New("ERROR"), "UTC", 99)
	ctx := context.Background()
	q := store.Queries()

	workspace, err := q.GetOrCreateWorkspace(ctx, -1001234567890, "Group", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertTopicBinding(ctx, workspace.ID, domain.TopicGoals, 14, "Цели"); err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertPendingInput(ctx, workspace.ID, 42, 13, domain.PendingRoutinePlan, map[string]any{
		"thread_id":         13,
		"prompt_message_id": 100,
	}); err != nil {
		t.Fatal(err)
	}

	answer, err := service.HandleUpdate(ctx, telegram.Update{Callback: &telegram.CallbackQuery{
		ID:   "goals-configure",
		From: telegram.User{ID: 42, Username: "igor", FirstName: "Игорь"},
		Data: "goals:configure",
		Message: &telegram.Message{
			MessageID:       200,
			MessageThreadID: 14,
			Chat:            telegram.Chat{ID: workspace.ChatID, Type: "supergroup", Title: "Group", IsForum: true},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if answer.Text != "" {
		t.Fatalf("callback answer = %q", answer.Text)
	}
	if len(fake.deleted) != 0 {
		t.Fatalf("goals configure should not delete routine prompt: %+v", fake.deleted)
	}
	routinePending, found, err := q.GetPendingInput(ctx, workspace.ID, 42, 13)
	if err != nil || !found || routinePending.Kind != domain.PendingRoutinePlan {
		t.Fatalf("routine pending should remain found=%v pending=%+v err=%v", found, routinePending, err)
	}
	goalsPending, found, err := q.GetPendingInput(ctx, workspace.ID, 42, 14)
	if err != nil || !found || goalsPending.Kind != domain.PendingSeasonalGoals || goalsPending.Payload["thread_id"] != float64(14) {
		t.Fatalf("unexpected goals pending found=%v pending=%+v err=%v", found, goalsPending, err)
	}
	if len(fake.sent) != 1 || !strings.Contains(fake.sent[0].Text, "Пришли сезонные цели") {
		t.Fatalf("expected one goals prompt, got %+v", fake.sent)
	}
}

func TestSeasonalGoalsSaveUsesConciseConfirmationWithoutEcho(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	fake := newFakeTelegram()
	service := bot.NewService(store, fake, logging.New("ERROR"), "UTC", 99)
	ctx := context.Background()
	q := store.Queries()

	workspace, err := q.GetOrCreateWorkspace(ctx, -1001234567890, "Group", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertTopicBinding(ctx, workspace.ID, domain.TopicGoals, 14, "Цели"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.HandleUpdate(ctx, telegram.Update{Callback: &telegram.CallbackQuery{
		ID:   "goals-configure",
		From: telegram.User{ID: 42, Username: "igor", FirstName: "Игорь"},
		Data: "goals:configure",
		Message: &telegram.Message{
			MessageID:       200,
			MessageThreadID: 14,
			Chat:            telegram.Chat{ID: workspace.ChatID, Type: "supergroup", Title: "Group", IsForum: true},
		},
	}}); err != nil {
		t.Fatal(err)
	}

	input := telegram.Message{
		MessageID:       301,
		MessageThreadID: 14,
		DateUnix:        time.Now().Unix(),
		From:            &telegram.User{ID: 42, Username: "igor", FirstName: "Игорь"},
		Chat:            telegram.Chat{ID: workspace.ChatID, Type: "supergroup", Title: "Group", IsForum: true},
		Text:            "1. Работа\n— Результат: получить предложение о работе",
	}
	if _, err := service.HandleUpdate(ctx, telegram.Update{Message: &input}); err != nil {
		t.Fatal(err)
	}
	if len(fake.sent) != 1 {
		t.Fatalf("goals save should not send a separate full goals card, sent=%+v", fake.sent)
	}
	edit, ok := fake.findEdit(1001)
	if !ok {
		t.Fatalf("confirmation edit missing: %+v", fake.edits)
	}
	if !strings.Contains(edit.Text, "Цели записаны") || strings.Contains(edit.Text, "получить предложение") {
		t.Fatalf("confirmation should be concise and not echo goals: %s", edit.Text)
	}
	if _, found, err := q.GetPendingInput(ctx, workspace.ID, 42, 14); err != nil || found {
		t.Fatalf("pending input found=%v err=%v", found, err)
	}
	if !fake.wasDeleted(301) {
		t.Fatalf("goals input message should be deleted after save: %+v", fake.deleted)
	}
	participant, err := q.RegisterParticipant(ctx, workspace.ID, 42, "igor", "Игорь")
	if err != nil {
		t.Fatal(err)
	}
	var goalCount int
	if err := store.Pool().QueryRow(ctx, `
SELECT count(*)
FROM seasonal_goal_sets
WHERE workspace_group_id = $1 AND participant_id = $2
`, workspace.ID, participant.ID).Scan(&goalCount); err != nil {
		t.Fatal(err)
	}
	if goalCount != 1 {
		t.Fatalf("goal sets = %d, want 1", goalCount)
	}
}

func TestNoticeDismissDeletesMessage(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	fake := newFakeTelegram()
	service := bot.NewService(store, fake, logging.New("ERROR"), "UTC", 99)

	answer, err := service.HandleUpdate(context.Background(), telegram.Update{Callback: &telegram.CallbackQuery{
		ID:   "notice-dismiss",
		From: telegram.User{ID: 42, Username: "igor", FirstName: "Игорь"},
		Data: "notice:dismiss",
		Message: &telegram.Message{
			MessageID:       777,
			MessageThreadID: 13,
			Chat:            telegram.Chat{ID: -1001234567890, Type: "supergroup", Title: "Group", IsForum: true},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if answer.Text != "" {
		t.Fatalf("dismiss should be silent, got %q", answer.Text)
	}
	if !fake.wasDeleted(777) {
		t.Fatalf("notice message was not deleted: %+v", fake.deleted)
	}
}

type fakeTelegram struct {
	createdTopics []string
	nextThreadID  int64
	nextMessageID int64
	edits         []telegram.EditMessageTextRequest
	sent          []telegram.SendMessageRequest
	deleted       []int64
}

func newFakeTelegram() *fakeTelegram {
	return &fakeTelegram{nextThreadID: 100, nextMessageID: 1000}
}

func (f *fakeTelegram) PollUpdates(context.Context, int64, int) ([]telegram.Update, error) {
	return nil, nil
}
func (f *fakeTelegram) AnswerCallbackQuery(context.Context, telegram.AnswerCallbackQueryRequest) error {
	return nil
}
func (f *fakeTelegram) SendMessage(_ context.Context, request telegram.SendMessageRequest) (telegram.Message, error) {
	f.nextMessageID++
	f.sent = append(f.sent, request)
	return telegram.Message{MessageID: f.nextMessageID, MessageThreadID: request.MessageThreadID, Chat: telegram.Chat{ID: request.ChatID, Type: "supergroup"}}, nil
}
func (f *fakeTelegram) EditMessageText(_ context.Context, request telegram.EditMessageTextRequest) error {
	f.edits = append(f.edits, request)
	return nil
}
func (f *fakeTelegram) DeleteMessage(_ context.Context, _ int64, messageID int64) error {
	f.deleted = append(f.deleted, messageID)
	return nil
}
func (f *fakeTelegram) PinChatMessage(context.Context, int64, int64) error {
	return nil
}
func (f *fakeTelegram) GetMe(context.Context) (telegram.User, error) {
	return telegram.User{ID: 99, IsBot: true}, nil
}
func (f *fakeTelegram) GetChat(_ context.Context, chatID int64) (telegram.Chat, error) {
	return telegram.Chat{ID: chatID, Type: "supergroup", Title: "Group", IsForum: true}, nil
}
func (f *fakeTelegram) GetChatMember(_ context.Context, _ int64, userID int64) (telegram.ChatMember, error) {
	return telegram.ChatMember{Status: "administrator", User: telegram.User{ID: userID}, CanManageTopics: true}, nil
}
func (f *fakeTelegram) CreateForumTopic(_ context.Context, request telegram.CreateForumTopicRequest) (telegram.ForumTopic, error) {
	f.nextThreadID++
	f.createdTopics = append(f.createdTopics, request.Name)
	return telegram.ForumTopic{MessageThreadID: f.nextThreadID, Name: request.Name}, nil
}
func (f *fakeTelegram) EditForumTopic(context.Context, telegram.EditForumTopicRequest) error {
	return nil
}

func (f *fakeTelegram) findEdit(messageID int64) (telegram.EditMessageTextRequest, bool) {
	for i := len(f.edits) - 1; i >= 0; i-- {
		if f.edits[i].MessageID == messageID {
			return f.edits[i], true
		}
	}
	return telegram.EditMessageTextRequest{}, false
}

func (f *fakeTelegram) wasDeleted(messageID int64) bool {
	for _, deleted := range f.deleted {
		if deleted == messageID {
			return true
		}
	}
	return false
}
