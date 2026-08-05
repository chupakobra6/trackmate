package bot

import (
	"context"
	"errors"
	"testing"

	"github.com/igor/trackmate/internal/storage/postgres"
	"github.com/igor/trackmate/internal/telegram"
)

func TestShowDailyEntryStatusReplaysOnOneMessage(t *testing.T) {
	tg := &lifecycleTelegram{}
	service := &Service{Telegram: tg}
	task := postgres.DailyTask{ID: 42}

	for i := 0; i < 3; i++ {
		if err := service.showDailyEntryStatus(context.Background(), -1001, 777, task); err != nil {
			t.Fatal(err)
		}
	}
	if len(tg.sent) != 0 {
		t.Fatalf("status transition sent duplicate messages: %+v", tg.sent)
	}
	if len(tg.edits) != 3 {
		t.Fatalf("edits=%d, want 3 callback replays", len(tg.edits))
	}
	for _, edit := range tg.edits {
		if edit.MessageID != 777 || edit.ReplyMarkup == nil || len(edit.ReplyMarkup.InlineKeyboard) != 1 {
			t.Fatalf("status transition did not stay on the callback message: %+v", edit)
		}
	}
}

func TestShowDailyEntryStatusTreatsTelegramReplayAsSuccess(t *testing.T) {
	tg := &lifecycleTelegram{editErr: errors.New("Bad Request: message is not modified")}
	service := &Service{Telegram: tg}
	if err := service.showDailyEntryStatus(context.Background(), -1001, 777, postgres.DailyTask{ID: 42}); err != nil {
		t.Fatalf("replayed callback should be idempotent: %v", err)
	}
}

func TestDismissTelegramMessageUsesTombstoneWhenDeleteFails(t *testing.T) {
	tg := &lifecycleTelegram{deleteErr: errors.New("message can't be deleted")}
	service := &Service{Telegram: tg}
	if err := service.dismissTelegramMessage(context.Background(), -1001, 777); err != nil {
		t.Fatal(err)
	}
	if len(tg.edits) != 1 || tg.edits[0].MessageID != 777 || tg.edits[0].ReplyMarkup == nil || len(tg.edits[0].ReplyMarkup.InlineKeyboard) != 0 {
		t.Fatalf("message did not become an inert tombstone: %+v", tg.edits)
	}
	if len(tg.markupEdits) != 0 {
		t.Fatalf("unexpected keyboard-only fallback: %+v", tg.markupEdits)
	}
}

func TestDismissTelegramMessageRemovesKeyboardAsLastFallback(t *testing.T) {
	tg := &lifecycleTelegram{
		deleteErr: errors.New("delete failed"),
		editErr:   errors.New("edit failed"),
	}
	service := &Service{Telegram: tg}
	if err := service.dismissTelegramMessage(context.Background(), -1001, 777); err != nil {
		t.Fatal(err)
	}
	if len(tg.markupEdits) != 1 || tg.markupEdits[0].MessageID != 777 {
		t.Fatalf("keyboard fallback missing: %+v", tg.markupEdits)
	}
}

func TestDismissTelegramMessageReturnsCombinedFailure(t *testing.T) {
	tg := &lifecycleTelegram{
		deleteErr: errors.New("delete failed"),
		editErr:   errors.New("edit failed"),
		markupErr: errors.New("markup failed"),
	}
	service := &Service{Telegram: tg}
	err := service.dismissTelegramMessage(context.Background(), -1001, 777)
	if err == nil || !errors.Is(err, tg.deleteErr) || !errors.Is(err, tg.editErr) || !errors.Is(err, tg.markupErr) {
		t.Fatalf("expected all Telegram failures, got %v", err)
	}
}

type lifecycleTelegram struct {
	sent        []telegram.SendMessageRequest
	edits       []telegram.EditMessageTextRequest
	markupEdits []telegram.EditMessageReplyMarkupRequest
	deleteErr   error
	editErr     error
	markupErr   error
}

func (f *lifecycleTelegram) PollUpdates(context.Context, int64, int) ([]telegram.Update, error) {
	return nil, nil
}

func (f *lifecycleTelegram) AnswerCallbackQuery(context.Context, telegram.AnswerCallbackQueryRequest) error {
	return nil
}

func (f *lifecycleTelegram) SendMessage(_ context.Context, request telegram.SendMessageRequest) (telegram.Message, error) {
	f.sent = append(f.sent, request)
	return telegram.Message{}, nil
}

func (f *lifecycleTelegram) EditMessageText(_ context.Context, request telegram.EditMessageTextRequest) error {
	f.edits = append(f.edits, request)
	return f.editErr
}

func (f *lifecycleTelegram) EditMessageReplyMarkup(_ context.Context, request telegram.EditMessageReplyMarkupRequest) error {
	f.markupEdits = append(f.markupEdits, request)
	return f.markupErr
}

func (f *lifecycleTelegram) DeleteMessage(context.Context, int64, int64) error {
	return f.deleteErr
}

func (f *lifecycleTelegram) PinChatMessage(context.Context, int64, int64) error {
	return nil
}

func (f *lifecycleTelegram) GetMe(context.Context) (telegram.User, error) {
	return telegram.User{}, nil
}

func (f *lifecycleTelegram) GetChat(context.Context, int64) (telegram.Chat, error) {
	return telegram.Chat{}, nil
}

func (f *lifecycleTelegram) GetChatMember(context.Context, int64, int64) (telegram.ChatMember, error) {
	return telegram.ChatMember{}, nil
}

func (f *lifecycleTelegram) CreateForumTopic(context.Context, telegram.CreateForumTopicRequest) (telegram.ForumTopic, error) {
	return telegram.ForumTopic{}, nil
}

func (f *lifecycleTelegram) EditForumTopic(context.Context, telegram.EditForumTopicRequest) error {
	return nil
}
