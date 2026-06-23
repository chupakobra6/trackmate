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
	if fake.sent[1].MessageThreadID != 20 {
		t.Fatalf("progress was not published into progress thread: %+v", fake.sent[1])
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
	plan, err := q.UpsertRoutinePlan(ctx, workspace.ID, participant.ID, participant.UserID, []string{"зарядка", "английский"})
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
	if _, err := q.UpsertSeasonalGoalSet(ctx, workspace.ID, participant.ID, participant.UserID, period, "Результат: оффер\nМетрика: 10 откликов"); err != nil {
		t.Fatal(err)
	}

	fake := &fakeTelegram{nextMessageID: 2000}
	runner := &worker.Runner{Store: store, TG: fake, Logger: logging.New("ERROR")}
	if err := runner.Tick(ctx, time.Date(2026, 6, 28, 20, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if !fake.hasSentToThread(30, "Рутина") {
		t.Fatalf("routine check-in not sent to routine topic: %+v", fake.sent)
	}
	if !fake.hasSentToThread(40, "Еженедельная проверка целей") {
		t.Fatalf("weekly goal review not sent to goals topic: %+v", fake.sent)
	}
	if fake.hasThread(20) {
		t.Fatalf("routine/goals worker should not publish progress events: %+v", fake.sent)
	}
	if pending, found, err := q.GetPendingInput(ctx, workspace.ID, participant.UserID); err != nil || !found || pending.Kind != domain.PendingGoalWeeklyReview {
		t.Fatalf("weekly pending found=%v pending=%+v err=%v", found, pending, err)
	}
	if err := q.ClearPendingInput(ctx, workspace.ID, participant.UserID); err != nil {
		t.Fatal(err)
	}

	if err := runner.Tick(ctx, time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if !fake.hasSentToThread(40, "Финал периода") {
		t.Fatalf("final goal review not sent to goals topic: %+v", fake.sent)
	}
}

type fakeTelegram struct {
	nextMessageID int64
	sent          []telegram.SendMessageRequest
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
func (f *fakeTelegram) EditMessageText(context.Context, telegram.EditMessageTextRequest) error {
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
