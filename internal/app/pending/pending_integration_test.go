package pending_test

import (
	"context"
	"testing"
	"time"

	apppending "github.com/igor/trackmate/internal/app/pending"
	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/telegram"
	"github.com/igor/trackmate/internal/testsupport"
)

func TestCleanupStaleInputsDeletesMessagesAndKeepsFreshTopic(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	ctx := context.Background()
	q := store.Queries()

	workspace, err := q.GetOrCreateWorkspace(ctx, -100888000556, "Group", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	participant, err := q.RegisterParticipant(ctx, workspace.ID, 42, "igor", "Igor")
	if err != nil {
		t.Fatal(err)
	}
	stale, err := q.UpsertPendingInput(ctx, workspace.ID, participant.UserID, 13, domain.PendingRoutinePlan, map[string]any{
		"prompt_message_id": 100,
		"user_message_ids":  []int64{201, 202},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertPendingInput(ctx, workspace.ID, participant.UserID, 14, domain.PendingSeasonalGoals, map[string]any{
		"prompt_message_id": 300,
	}); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 6, 24, 20, 0, 0, 0, time.UTC)
	if _, err := store.Pool().Exec(ctx, `UPDATE pending_inputs SET created_at = $2 WHERE id = $1`, stale.ID, now.Add(-25*time.Hour)); err != nil {
		t.Fatal(err)
	}

	fake := &fakeTelegram{}
	if err := apppending.CleanupStaleInputs(ctx, store, fake, now); err != nil {
		t.Fatal(err)
	}
	if _, found, err := q.GetPendingInput(ctx, workspace.ID, participant.UserID, 13); err != nil || found {
		t.Fatalf("stale pending should be removed found=%v err=%v", found, err)
	}
	if _, found, err := q.GetPendingInput(ctx, workspace.ID, participant.UserID, 14); err != nil || !found {
		t.Fatalf("fresh pending should remain found=%v err=%v", found, err)
	}
	for _, messageID := range []int64{100, 201, 202} {
		if !fake.wasDeleted(messageID) {
			t.Fatalf("message %d was not deleted: %+v", messageID, fake.deleted)
		}
	}
	if fake.wasDeleted(300) {
		t.Fatalf("fresh prompt should not be deleted: %+v", fake.deleted)
	}
}

type fakeTelegram struct {
	deleted []int64
}

func (f *fakeTelegram) PollUpdates(context.Context, int64, int) ([]telegram.Update, error) {
	return nil, nil
}
func (f *fakeTelegram) AnswerCallbackQuery(context.Context, telegram.AnswerCallbackQueryRequest) error {
	return nil
}
func (f *fakeTelegram) SendMessage(context.Context, telegram.SendMessageRequest) (telegram.Message, error) {
	return telegram.Message{}, nil
}
func (f *fakeTelegram) EditMessageText(context.Context, telegram.EditMessageTextRequest) error {
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

func (f *fakeTelegram) wasDeleted(messageID int64) bool {
	for _, deleted := range f.deleted {
		if deleted == messageID {
			return true
		}
	}
	return false
}
