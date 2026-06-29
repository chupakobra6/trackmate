package routine_test

import (
	"context"
	"strings"
	"testing"
	"time"

	approutine "github.com/igor/trackmate/internal/app/routine"
	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/telegram"
	"github.com/igor/trackmate/internal/testsupport"
)

func TestDispatchDueCheckinsAndRefreshLeaderboard(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	ctx := context.Background()
	q := store.Queries()

	workspace, err := q.GetOrCreateWorkspace(ctx, -100888000444, "Group", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertTopicBinding(ctx, workspace.ID, domain.TopicRoutine, 30, "Рутины"); err != nil {
		t.Fatal(err)
	}
	introID := int64(900)
	if err := q.SetTopicMessages(ctx, workspace.ID, domain.TopicRoutine, &introID, nil, false, false); err != nil {
		t.Fatal(err)
	}
	participant, err := q.RegisterParticipant(ctx, workspace.ID, 42, "igor", "Igor")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := q.UpsertRoutinePlan(ctx, workspace.ID, participant.ID, participant.UserID, []string{"зарядка", "йога"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Pool().Exec(ctx, `UPDATE routine_plans SET created_at = $2 WHERE id = $1`, plan.ID, time.Date(2026, 6, 28, 8, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}

	fake := &fakeTelegram{nextMessageID: 2000}
	now := time.Date(2026, 6, 28, 20, 0, 0, 0, time.UTC)
	if err := approutine.DispatchDueCheckins(ctx, store, fake, nil, now); err != nil {
		t.Fatal(err)
	}
	if len(fake.sent) != 1 || fake.sent[0].MessageThreadID != 30 || !strings.Contains(fake.sent[0].Text, "Рутина") {
		t.Fatalf("unexpected routine dispatch: %+v", fake.sent)
	}
	checkin, found, err := q.GetRoutineCheckinForDate(ctx, workspace.ID, participant.ID, now)
	if err != nil || !found {
		t.Fatalf("checkin found=%v err=%v", found, err)
	}
	if checkin.CardMessageID == nil || *checkin.CardMessageID != 2001 {
		t.Fatalf("checkin card message was not stored: %+v", checkin)
	}

	if err := approutine.RefreshLeaderboard(ctx, q, fake, workspace, workspace.ChatID, now); err != nil {
		t.Fatal(err)
	}
	edit, ok := fake.findEdit(introID)
	if !ok || !strings.Contains(edit.Text, "Таблица рутин") {
		t.Fatalf("routine table intro was not edited: found=%v edit=%+v", ok, edit)
	}
}

func TestRunCheckinTransitionsRemindsAndAutoCloses(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	ctx := context.Background()
	q := store.Queries()

	workspace, err := q.GetOrCreateWorkspace(ctx, -100888000445, "Group", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertTopicBinding(ctx, workspace.ID, domain.TopicRoutine, 30, "Рутины"); err != nil {
		t.Fatal(err)
	}
	introID := int64(900)
	if err := q.SetTopicMessages(ctx, workspace.ID, domain.TopicRoutine, &introID, nil, false, false); err != nil {
		t.Fatal(err)
	}
	participant, err := q.RegisterParticipant(ctx, workspace.ID, 42, "igor", "Igor")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := q.UpsertRoutinePlan(ctx, workspace.ID, participant.ID, participant.UserID, []string{"зарядка", "йога"})
	if err != nil {
		t.Fatal(err)
	}
	checkin, err := q.GetOrCreateRoutineCheckin(ctx, plan, time.Date(2026, 6, 28, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if err := q.SetRoutineCheckinCardMessageID(ctx, checkin.ID, 2100, 30); err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertPendingInput(ctx, workspace.ID, participant.UserID, 30, domain.PendingRoutineReason, map[string]any{
		"checkin_id":        checkin.ID,
		"prompt_message_id": 2200,
		"user_message_ids":  []int64{2201},
	}); err != nil {
		t.Fatal(err)
	}

	fake := &fakeTelegram{nextMessageID: 3000}
	if err := approutine.RunCheckinTransitions(ctx, store, fake, nil, time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if len(fake.sent) != 1 || !strings.Contains(fake.sent[0].Text, "еще не закрыта") || fake.sent[0].ReplyToMessageID != 2100 || fake.sent[0].ReplyMarkup == nil {
		t.Fatalf("unexpected reminder send: %+v", fake.sent)
	}
	reminded, found, err := q.GetRoutineCheckin(ctx, checkin.ID)
	if err != nil || !found || reminded.ReminderSentAt == nil || reminded.ReminderMessageID == nil {
		t.Fatalf("reminder was not stored found=%v checkin=%+v err=%v", found, reminded, err)
	}

	if err := approutine.RunCheckinTransitions(ctx, store, fake, nil, time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	closed, found, err := q.GetRoutineCheckin(ctx, checkin.ID)
	if err != nil || !found {
		t.Fatalf("closed checkin found=%v err=%v", found, err)
	}
	if closed.CompletedAt == nil || closed.AutoFailedAt == nil {
		t.Fatalf("checkin was not auto-closed: %+v", closed)
	}
	for _, item := range closed.Items {
		if item.Status == nil || *item.Status != domain.RoutineItemFailed {
			t.Fatalf("item was not failed: %+v", item)
		}
	}
	if _, found, err := q.GetPendingInput(ctx, workspace.ID, participant.UserID, 30); err != nil || found {
		t.Fatalf("routine pending should be cleared found=%v err=%v", found, err)
	}
	if reminded.ReminderMessageID == nil || !fake.wasDeleted(*reminded.ReminderMessageID) {
		t.Fatalf("routine reminder should be deleted on auto-close, deleted=%+v reminder=%+v", fake.deleted, reminded.ReminderMessageID)
	}
	for _, messageID := range []int64{2100, 2200, 2201} {
		if !fake.wasDeleted(messageID) {
			t.Fatalf("routine auto-close should delete message %d, deleted=%+v", messageID, fake.deleted)
		}
	}
	if fake.findEditCount(2100) != 0 {
		t.Fatalf("routine card should not be edited on auto-close, edits=%+v", fake.edits)
	}
	if len(fake.sent) != 1 {
		t.Fatalf("auto-close should not send extra routine notices: %+v", fake.sent)
	}
	if tableEdit, ok := fake.findEdit(introID); !ok || !strings.Contains(tableEdit.Text, "Таблица рутин") {
		t.Fatalf("routine table refresh missing: found=%v edit=%+v", ok, tableEdit)
	}
}

type fakeTelegram struct {
	nextMessageID int64
	sent          []telegram.SendMessageRequest
	edits         []telegram.EditMessageTextRequest
	deleted       []int64
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

func (f *fakeTelegram) findEdit(messageID int64) (telegram.EditMessageTextRequest, bool) {
	for _, edit := range f.edits {
		if edit.MessageID == messageID {
			return edit, true
		}
	}
	return telegram.EditMessageTextRequest{}, false
}

func (f *fakeTelegram) findEditCount(messageID int64) int {
	var count int
	for _, edit := range f.edits {
		if edit.MessageID == messageID {
			count++
		}
	}
	return count
}

func (f *fakeTelegram) wasDeleted(messageID int64) bool {
	for _, deleted := range f.deleted {
		if deleted == messageID {
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
