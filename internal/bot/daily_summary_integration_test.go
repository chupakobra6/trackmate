package bot_test

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/igor/trackmate/internal/bot"
	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/logging"
	"github.com/igor/trackmate/internal/telegram"
	"github.com/igor/trackmate/internal/testsupport"
)

func TestTodayAddAfterCutoffCreatesAndClosesDailySummary(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	ctx := context.Background()
	q := store.Queries()
	workspace, err := q.GetOrCreateWorkspace(ctx, -1001234567891, "Group", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertTopicBinding(ctx, workspace.ID, domain.TopicToday, 10, "Сегодня"); err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertTopicBinding(ctx, workspace.ID, domain.TopicProgress, 11, "Прогресс"); err != nil {
		t.Fatal(err)
	}
	cutoff := time.Date(2026, 7, 12, 20, 0, 0, 0, time.UTC)
	if err := q.SetClockOverride(ctx, &cutoff); err != nil {
		t.Fatal(err)
	}

	fake := newFakeTelegram()
	service := bot.NewService(store, fake, logging.New("ERROR"), "UTC", 99)
	user := telegram.User{ID: 42, Username: "igor", FirstName: "Игорь"}
	participant, err := q.RegisterParticipant(ctx, workspace.ID, user.ID, user.Username, user.FirstName)
	if err != nil {
		t.Fatal(err)
	}
	control := telegram.Message{MessageID: 100, MessageThreadID: 10, Chat: telegram.Chat{ID: workspace.ChatID, Type: "supergroup", Title: "Group", IsForum: true}}
	answer, err := service.HandleUpdate(ctx, telegram.Update{Callback: &telegram.CallbackQuery{ID: "summary-add", From: user, Data: "today:add", Message: &control}})
	if err != nil || answer.Text != "" {
		t.Fatalf("summary add answer=%+v err=%v", answer, err)
	}
	if len(fake.sent) != 1 {
		t.Fatalf("summary card sends=%d, want 1", len(fake.sent))
	}
	if !containsAll(fake.sent[0].Text, "🏁 <b>Итог дня</b>", "<b>Статус:</b> ожидается") {
		t.Fatalf("unexpected summary card: %+v", fake.sent[0])
	}
	if fake.sent[0].ReplyMarkup == nil || len(fake.sent[0].ReplyMarkup.InlineKeyboard) != 1 {
		t.Fatalf("summary card keyboard missing: %+v", fake.sent[0].ReplyMarkup)
	}
	buttons := fake.sent[0].ReplyMarkup.InlineKeyboard[0]
	if len(buttons) != 3 || buttons[0].Text != "✅ Хорошо" || buttons[1].Text != "🔸 Средне" || buttons[2].Text != "❌ Плохо" {
		t.Fatalf("unexpected summary statuses: %+v", buttons)
	}

	date := time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC)
	summary, found, err := q.GetTaskForDate(ctx, workspace.ID, participant.ID, date)
	if err != nil || !found || summary.Kind != domain.DailyEntrySummary || summary.TodayCardMessageID == nil {
		t.Fatalf("summary found=%v summary=%+v err=%v", found, summary, err)
	}

	card := telegram.Message{MessageID: *summary.TodayCardMessageID, MessageThreadID: 10, Chat: control.Chat}
	answer, err = service.HandleUpdate(ctx, telegram.Update{Callback: &telegram.CallbackQuery{ID: "summary-medium", From: user, Data: "task:status:" + strconv.FormatInt(summary.ID, 10) + ":partial", Message: &card}})
	if err != nil || answer.Text != "" {
		t.Fatalf("summary status answer=%+v err=%v", answer, err)
	}
	promptEdit, ok := fake.findEdit(*summary.TodayCardMessageID)
	if !ok || !strings.Contains(promptEdit.Text, "Напиши итог дня одним сообщением") || promptEdit.ReplyMarkup == nil || len(promptEdit.ReplyMarkup.InlineKeyboard) != 0 {
		t.Fatalf("summary prompt edit mismatch: found=%v edit=%+v", ok, promptEdit)
	}
	pending, found, err := q.GetPendingInput(ctx, workspace.ID, user.ID, 10)
	if err != nil || !found || pending.Kind != domain.PendingDailySummaryReport {
		t.Fatalf("summary pending found=%v pending=%+v err=%v", found, pending, err)
	}

	report := telegram.Message{MessageID: 301, MessageThreadID: 10, DateUnix: time.Now().Unix(), From: &user, Chat: control.Chat, Text: "Разобрал ключевой вопрос и не распылялся."}
	if _, err := service.HandleUpdate(ctx, telegram.Update{Message: &report}); err != nil {
		t.Fatal(err)
	}
	updated, found, err := q.GetTask(ctx, summary.ID)
	if err != nil || !found || updated.Status != domain.DailyTaskPartial || updated.ReportText == nil {
		t.Fatalf("closed summary found=%v summary=%+v err=%v", found, updated, err)
	}
	if *updated.ReportText != report.Text {
		t.Fatalf("summary report = %q", *updated.ReportText)
	}
	if fake.wasDeleted(*summary.TodayCardMessageID) {
		t.Fatalf("persistent summary card must not be deleted: %+v", fake.deleted)
	}
	closedEdit, ok := fake.findEdit(*summary.TodayCardMessageID)
	if !ok || !containsAll(closedEdit.Text, "подвёл итог среднего дня", "<b>Итог:</b>", report.Text) {
		t.Fatalf("closed summary card mismatch: found=%v edit=%+v", ok, closedEdit)
	}
	if strings.Contains(closedEdit.Text, "<b>Оценка:</b>") || strings.Contains(closedEdit.Text, "Частично") {
		t.Fatalf("summary card has obsolete score wording: %s", closedEdit.Text)
	}
	if closedEdit.ReplyMarkup == nil || len(closedEdit.ReplyMarkup.InlineKeyboard) != 0 {
		t.Fatalf("summary card keyboard was not removed: %+v", closedEdit.ReplyMarkup)
	}
	events, err := q.ListPendingProgressEvents(ctx)
	if err != nil || len(events) != 1 || events[0].EventType != domain.ProgressDailySummaryClosed {
		t.Fatalf("summary progress events=%+v err=%v", events, err)
	}
	if err := q.MarkProgressEventPublished(ctx, events[0].ID, 501, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	editedReport := report
	editedReport.Text = "Разобрал ключевой вопрос и закрепил выводы."
	if _, err := service.HandleUpdate(ctx, telegram.Update{EditedMessage: &editedReport}); err != nil {
		t.Fatal(err)
	}
	editedCard, ok := fake.findEdit(*summary.TodayCardMessageID)
	if !ok || !strings.Contains(editedCard.Text, editedReport.Text) || strings.Contains(editedCard.Text, report.Text) {
		t.Fatalf("summary card did not sync source edit: found=%v edit=%+v", ok, editedCard)
	}
	editedProgress, ok := fake.findEdit(501)
	if !ok || !containsAll(editedProgress.Text, "подвёл итог среднего дня", editedReport.Text) || strings.Contains(editedProgress.Text, report.Text) {
		t.Fatalf("summary progress did not sync source edit: found=%v edit=%+v", ok, editedProgress)
	}
}

func TestTodayAddBeforeCutoffKeepsTaskPrompt(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	ctx := context.Background()
	q := store.Queries()
	workspace, err := q.GetOrCreateWorkspace(ctx, -1001234567892, "Group", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertTopicBinding(ctx, workspace.ID, domain.TopicToday, 10, "Сегодня"); err != nil {
		t.Fatal(err)
	}
	before := time.Date(2026, 7, 12, 19, 59, 0, 0, time.UTC)
	if err := q.SetClockOverride(ctx, &before); err != nil {
		t.Fatal(err)
	}
	fake := newFakeTelegram()
	service := bot.NewService(store, fake, logging.New("ERROR"), "UTC", 99)
	user := telegram.User{ID: 42, Username: "igor", FirstName: "Игорь"}
	control := telegram.Message{MessageID: 100, MessageThreadID: 10, Chat: telegram.Chat{ID: workspace.ChatID, Type: "supergroup", Title: "Group", IsForum: true}}
	if _, err := service.HandleUpdate(ctx, telegram.Update{Callback: &telegram.CallbackQuery{ID: "task-add", From: user, Data: "today:add", Message: &control}}); err != nil {
		t.Fatal(err)
	}
	if len(fake.sent) != 1 || !strings.Contains(fake.sent[0].Text, "Напиши главную задачу дня одним сообщением") {
		t.Fatalf("task prompt missing before cutoff: %+v", fake.sent)
	}
	pending, found, err := q.GetPendingInput(ctx, workspace.ID, user.ID, 10)
	if err != nil || !found || pending.Kind != domain.PendingDailyTaskText {
		t.Fatalf("task pending found=%v pending=%+v err=%v", found, pending, err)
	}
}
