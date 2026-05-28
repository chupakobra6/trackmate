package bot_test

import (
	"context"
	"testing"
	"time"

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
	task, created, err := q.CreateDailyTask(ctx, workspace.ID, participant.ID, participant.UserID, time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC), "Приложить фотоотчет")
	if err != nil || !created {
		t.Fatalf("task created=%v err=%v", created, err)
	}
	if err := q.SetDailyTaskCardMessageID(ctx, task.ID, 100); err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertPendingInput(ctx, workspace.ID, participant.UserID, domain.PendingDailyTaskReport, map[string]any{
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
		Caption:         "Фотоотчет: задача закрыта двумя изображениями.",
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
	if updated.Status != domain.DailyTaskDone || updated.ReportText == nil || *updated.ReportText != "Фотоотчет: задача закрыта двумя изображениями." {
		t.Fatalf("unexpected report state: %+v", updated)
	}
	if _, found, err := q.GetPendingInput(ctx, workspace.ID, participant.UserID); err != nil || found {
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
