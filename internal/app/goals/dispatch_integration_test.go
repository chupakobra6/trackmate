package goals_test

import (
	"context"
	"strings"
	"testing"
	"time"

	appgoals "github.com/igor/trackmate/internal/app/goals"
	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/telegram"
	"github.com/igor/trackmate/internal/testsupport"
)

func TestDispatchWeeklyAndFinalReviews(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	ctx := context.Background()
	q := store.Queries()

	workspace, err := q.GetOrCreateWorkspace(ctx, -100888000555, "Group", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertTopicBinding(ctx, workspace.ID, domain.TopicGoals, 40, "Цели"); err != nil {
		t.Fatal(err)
	}
	participant, err := q.RegisterParticipant(ctx, workspace.ID, 42, "igor", "Igor")
	if err != nil {
		t.Fatal(err)
	}
	period := domain.GoalPeriod{
		Key:      "summer-2026",
		Title:    "Лето 2026",
		StartsOn: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		EndsOn:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
	}
	sourceMessageID := int64(501)
	sourceThreadID := int64(40)
	if _, err := q.UpsertSeasonalGoalSet(ctx, workspace.ID, participant.ID, participant.UserID, period, "Результат: предложение о работе\nМетрика: 10 откликов", &sourceMessageID, &sourceThreadID); err != nil {
		t.Fatal(err)
	}

	fake := &fakeTelegram{nextMessageID: 3000}
	if err := appgoals.DispatchWeeklyReviews(ctx, store, fake, time.Date(2026, 6, 28, 20, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if !fake.hasSentToThread(40, "Обзор целей") {
		t.Fatalf("weekly review was not sent to goals topic: %+v", fake.sent)
	}
	if fake.hasSentToThread(40, "Результат: предложение") {
		t.Fatalf("weekly review should not echo full goals: %+v", fake.sent)
	}
	if !fake.hasSentToThread(40, `https://t.me/c/888000555/501?thread=40`) {
		t.Fatalf("weekly review should link to source goals message: %+v", fake.sent)
	}
	if pending, found, err := q.GetPendingInput(ctx, workspace.ID, participant.UserID, 40); err != nil || !found || pending.Kind != domain.PendingGoalWeeklyReview {
		t.Fatalf("weekly pending found=%v pending=%+v err=%v", found, pending, err)
	}
	if err := q.ClearPendingInput(ctx, workspace.ID, participant.UserID, 40); err != nil {
		t.Fatal(err)
	}

	if err := appgoals.DispatchFinalReviews(ctx, store, fake, time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if !fake.hasSentToThread(40, "Итог периода") {
		t.Fatalf("final review was not sent to goals topic: %+v", fake.sent)
	}
}

func (f *fakeTelegram) hasSentToThread(threadID int64, text string) bool {
	for _, sent := range f.sent {
		if sent.MessageThreadID == threadID && strings.Contains(sent.Text, text) {
			return true
		}
	}
	return false
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
