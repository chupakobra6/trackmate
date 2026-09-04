package worker_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/logging"
	"github.com/igor/trackmate/internal/telegram"
	"github.com/igor/trackmate/internal/testsupport"
	"github.com/igor/trackmate/internal/worker"
)

func TestWorkerTransitionsDispatchesAlertAndPublishesProgress(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	ctx := context.Background()
	q := store.Queries()
	workspace, err := q.GetOrCreateWorkspace(ctx, -100777000111, "Group", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertTopicBinding(ctx, workspace.ID, domain.TopicToday, 10, "Сегодня"); err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertTopicBinding(ctx, workspace.ID, domain.TopicProgress, 20, "Прогресс"); err != nil {
		t.Fatal(err)
	}
	participant, err := q.RegisterParticipant(ctx, workspace.ID, 42, "igor", "Igor")
	if err != nil {
		t.Fatal(err)
	}
	task, created, err := q.CreateDailyTask(ctx, workspace.ID, participant.ID, participant.UserID, time.Date(2026, 5, 27, 0, 0, 0, 0, time.UTC), "Task", 200, 10)
	if err != nil || !created {
		t.Fatalf("task created=%v err=%v", created, err)
	}
	if err := q.SetDailyTaskCardMessageID(ctx, task.ID, 555); err != nil {
		t.Fatal(err)
	}
	fake := &fakeTelegram{nextMessageID: 1000}
	runner := &worker.Runner{Store: store, TG: fake, Logger: logging.New("ERROR")}
	if err := runner.Tick(ctx, time.Date(2026, 5, 28, 12, 1, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	refreshed, found, err := q.GetTask(ctx, task.ID)
	if err != nil || !found {
		t.Fatalf("task found=%v err=%v", found, err)
	}
	if refreshed.Status != domain.DailyTaskFailed {
		t.Fatalf("expected failed task, got %s", refreshed.Status)
	}
	if len(fake.sent) != 2 {
		t.Fatalf("expected alert and progress sends, got %d", len(fake.sent))
	}
	if fake.sent[0].MessageThreadID != 10 || fake.sent[0].ReplyToMessageID != 555 {
		t.Fatalf("alert was not sent into today thread as a task reply: %+v", fake.sent[0])
	}
	if fake.sent[0].DisableNotification {
		t.Fatalf("missed-task alert should notify: %+v", fake.sent[0])
	}
	if fake.sent[1].MessageThreadID != 20 {
		t.Fatalf("progress was not published into progress thread: %+v", fake.sent[1])
	}
	if !fake.sent[1].DisableNotification {
		t.Fatalf("progress should be silent: %+v", fake.sent[1])
	}
	if len(fake.edits) != 1 || fake.edits[0].MessageID != 555 || !strings.Contains(fake.edits[0].Text, "не выполнил задачу дня") || fake.edits[0].ReplyMarkup == nil || len(fake.edits[0].ReplyMarkup.InlineKeyboard) != 0 {
		t.Fatalf("task card was not auto-closed without buttons: %+v", fake.edits)
	}
}

func TestWorkerTransitionsDailySummaryWithOwnAlertsAndProgress(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	ctx := context.Background()
	q := store.Queries()
	workspace, err := q.GetOrCreateWorkspace(ctx, -100777000112, "Group", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertTopicBinding(ctx, workspace.ID, domain.TopicToday, 10, "Сегодня"); err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertTopicBinding(ctx, workspace.ID, domain.TopicProgress, 20, "Прогресс"); err != nil {
		t.Fatal(err)
	}
	participant, err := q.RegisterParticipant(ctx, workspace.ID, 42, "igor", "Игорь")
	if err != nil {
		t.Fatal(err)
	}
	summary, created, err := q.CreateDailySummary(ctx, workspace.ID, participant.ID, participant.UserID, time.Date(2026, 5, 27, 0, 0, 0, 0, time.UTC))
	if err != nil || !created {
		t.Fatalf("summary created=%v err=%v", created, err)
	}
	if err := q.SetDailyTaskCardMessageID(ctx, summary.ID, 555); err != nil {
		t.Fatal(err)
	}
	fake := &fakeTelegram{nextMessageID: 1100}
	runner := &worker.Runner{Store: store, TG: fake, Logger: logging.New("ERROR")}
	if err := runner.Tick(ctx, time.Date(2026, 5, 28, 0, 1, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	awaiting, found, err := q.GetTask(ctx, summary.ID)
	if err != nil || !found || awaiting.Status != domain.DailyTaskAwaitingReport {
		t.Fatalf("awaiting summary found=%v summary=%+v err=%v", found, awaiting, err)
	}
	if len(fake.sent) != 1 || !strings.Contains(fake.sent[0].Text, "итог дня ещё не записан") {
		t.Fatalf("summary pending alert mismatch: %+v", fake.sent)
	}

	if err := runner.Tick(ctx, time.Date(2026, 5, 28, 12, 1, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	failed, found, err := q.GetTask(ctx, summary.ID)
	if err != nil || !found || failed.Status != domain.DailyTaskFailed {
		t.Fatalf("failed summary found=%v summary=%+v err=%v", found, failed, err)
	}
	if len(fake.sent) != 3 {
		t.Fatalf("summary sends=%d, want 3: %+v", len(fake.sent), fake.sent)
	}
	if !strings.Contains(fake.sent[1].Text, "Итог дня отмечен как невыполненный") {
		t.Fatalf("summary failed alert mismatch: %+v", fake.sent[1])
	}
	if !strings.Contains(fake.sent[2].Text, "не подвёл итог дня вовремя") {
		t.Fatalf("summary progress mismatch: %+v", fake.sent[2])
	}
	if len(fake.edits) != 1 || fake.edits[0].MessageID != 555 || !strings.Contains(fake.edits[0].Text, "не подвёл итог дня") || fake.edits[0].ReplyMarkup == nil || len(fake.edits[0].ReplyMarkup.InlineKeyboard) != 0 {
		t.Fatalf("summary card was not auto-closed: %+v", fake.edits)
	}
}

func TestWorkerDispatchesRoutineAndGoalPromptsToOwnTopics(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	ctx := context.Background()
	q := store.Queries()
	workspace, err := q.GetOrCreateWorkspace(ctx, -100777000222, "Group", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertTopicBinding(ctx, workspace.ID, domain.TopicProgress, 20, "Прогресс"); err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertTopicBinding(ctx, workspace.ID, domain.TopicRoutine, 30, "Рутины"); err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertTopicBinding(ctx, workspace.ID, domain.TopicGoals, 40, "Цели"); err != nil {
		t.Fatal(err)
	}
	participant, err := q.RegisterParticipant(ctx, workspace.ID, 42, "igor", "Igor")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := q.UpsertRoutinePlan(ctx, workspace.ID, participant.ID, participant.UserID, []string{"зарядка", "английский"}, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Pool().Exec(ctx, `UPDATE routine_plans SET created_at = $2 WHERE id = $1`, plan.ID, time.Date(2026, 6, 27, 8, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	period := domain.GoalPeriod{
		Key:      "summer-2026",
		Title:    "Лето 2026",
		StartsOn: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		EndsOn:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
	}
	if _, err := q.UpsertSeasonalGoalSet(ctx, workspace.ID, participant.ID, participant.UserID, period, "Результат: предложение о работе\nМетрика: 10 откликов", nil, nil); err != nil {
		t.Fatal(err)
	}

	fake := &fakeTelegram{nextMessageID: 2000}
	runner := &worker.Runner{Store: store, TG: fake, Logger: logging.New("ERROR")}
	if err := runner.Tick(ctx, time.Date(2026, 6, 28, 20, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if !fake.hasSentToThread(30, "Рутина") {
		t.Fatalf("routine check-in should be sent after the next 08:00 window: %+v", fake.sent)
	}
	if err := runner.Tick(ctx, time.Date(2026, 6, 29, 8, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if !fake.hasSentToThread(30, "Рутина") {
		t.Fatalf("routine check-in not sent to routine topic: %+v", fake.sent)
	}
	if !fake.hasSentToThread(40, "Вопросы по целям") {
		t.Fatalf("weekly goal review not sent to goals topic: %+v", fake.sent)
	}
	if fake.hasThread(20) {
		t.Fatalf("routine/goals worker should not publish progress events: %+v", fake.sent)
	}
	if pending, found, err := q.GetPendingInput(ctx, workspace.ID, participant.UserID, 40); err != nil || !found || pending.Kind != domain.PendingGoalWeeklyReview {
		t.Fatalf("weekly pending found=%v pending=%+v err=%v", found, pending, err)
	}
	if err := q.ClearPendingInput(ctx, workspace.ID, participant.UserID, 40); err != nil {
		t.Fatal(err)
	}

	if err := runner.Tick(ctx, time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if !fake.hasSentToThread(40, "Итог периода") {
		t.Fatalf("final goal review not sent to goals topic: %+v", fake.sent)
	}
}

type fakeTelegram struct {
	nextMessageID int64
	sent          []telegram.SendMessageRequest
	edits         []telegram.EditMessageTextRequest
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
func (f *fakeTelegram) DeleteMessage(context.Context, int64, int64) error {
	return nil
}
func (f *fakeTelegram) PinChatMessage(context.Context, int64, int64) error {
	return nil
}
func (f *fakeTelegram) GetMe(context.Context) (telegram.User, error) {
	return telegram.User{}, nil
}
func (f *fakeTelegram) GetChat(context.Context, int64) (telegram.Chat, error) {
	return telegram.Chat{}, nil
}
func (f *fakeTelegram) GetChatMember(context.Context, int64, int64) (telegram.ChatMember, error) {
	return telegram.ChatMember{}, nil
}
func (f *fakeTelegram) CreateForumTopic(context.Context, telegram.CreateForumTopicRequest) (telegram.ForumTopic, error) {
	return telegram.ForumTopic{}, nil
}
func (f *fakeTelegram) EditForumTopic(context.Context, telegram.EditForumTopicRequest) error {
	return nil
}

func (f *fakeTelegram) hasThread(threadID int64) bool {
	for _, sent := range f.sent {
		if sent.MessageThreadID == threadID {
			return true
		}
	}
	return false
}

func (f *fakeTelegram) hasSentToThread(threadID int64, text string) bool {
	for _, sent := range f.sent {
		if sent.MessageThreadID == threadID && strings.Contains(sent.Text, text) {
			return true
		}
	}
	return false
}
