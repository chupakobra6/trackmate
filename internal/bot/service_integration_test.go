package bot_test

import (
	"context"
	"testing"

	"github.com/igor/trackmate/internal/bot"
	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/logging"
	"github.com/igor/trackmate/internal/telegram"
	"github.com/igor/trackmate/internal/testsupport"
)

func TestSetupCreatesOnlyTodayAndProgress(t *testing.T) {
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
	if len(fake.createdTopics) != 2 {
		t.Fatalf("expected two active topics, got %v", fake.createdTopics)
	}
	if fake.createdTopics[0] != "Сегодня" || fake.createdTopics[1] != "Прогресс" {
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
}

type fakeTelegram struct {
	createdTopics []string
	nextThreadID  int64
	nextMessageID int64
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
