package routine_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	approutine "github.com/igor/trackmate/internal/app/routine"
	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/storage/postgres"
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
	plan, err := q.UpsertRoutinePlan(ctx, workspace.ID, participant.ID, participant.UserID, []string{"зарядка", "йога"}, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Pool().Exec(ctx, `UPDATE routine_plans SET created_at = $2 WHERE id = $1`, plan.ID, time.Date(2026, 6, 28, 8, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}

	fake := &fakeTelegram{nextMessageID: 2000}
	now := time.Date(2026, 6, 29, 8, 0, 0, 0, time.UTC)
	if err := approutine.DispatchDueCheckins(ctx, store, fake, nil, now); err != nil {
		t.Fatal(err)
	}
	if len(fake.sent) != 1 || fake.sent[0].MessageThreadID != 30 || !strings.Contains(fake.sent[0].Text, "Рутина Игоря за воскресенье, 28 июня") || !fake.sent[0].DisableNotification {
		t.Fatalf("unexpected routine dispatch: %+v", fake.sent)
	}
	checkin, found, err := q.GetRoutineCheckinForDate(ctx, workspace.ID, participant.ID, time.Date(2026, 6, 28, 0, 0, 0, 0, time.UTC))
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
	if !ok || !strings.Contains(edit.Text, "Лидерборд") {
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
	participant, err := q.RegisterParticipant(ctx, workspace.ID, 42, "igor", "Игорь")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := q.UpsertRoutinePlan(ctx, workspace.ID, participant.ID, participant.UserID, []string{"зарядка", "йога"}, 2000, 30)
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
	if err := approutine.RunCheckinTransitions(ctx, store, fake, nil, time.Date(2026, 6, 29, 19, 59, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if len(fake.sent) != 0 {
		t.Fatalf("routine should not remind before 20:00: %+v", fake.sent)
	}
	if err := approutine.RunCheckinTransitions(ctx, store, fake, nil, time.Date(2026, 6, 29, 20, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if len(fake.sent) != 1 || !strings.Contains(fake.sent[0].Text, "Жду ответы до полуночи") || strings.Contains(fake.sent[0].Text, "12:00") || fake.sent[0].ReplyToMessageID != 2100 || fake.sent[0].ReplyMarkup == nil || fake.sent[0].DisableNotification {
		t.Fatalf("unexpected reminder send: %+v", fake.sent)
	}
	reminded, found, err := q.GetRoutineCheckin(ctx, checkin.ID)
	if err != nil || !found || reminded.ReminderSentAt == nil || reminded.ReminderMessageID == nil {
		t.Fatalf("reminder was not stored found=%v checkin=%+v err=%v", found, reminded, err)
	}

	if err := approutine.RunCheckinTransitions(ctx, store, fake, nil, time.Date(2026, 6, 29, 23, 59, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if fake.wasDeleted(2100) {
		t.Fatalf("routine should not auto-close before midnight, deleted=%+v", fake.deleted)
	}
	fake.sendError = errors.New("request timeout")
	if err := approutine.RunCheckinTransitions(ctx, store, fake, nil, time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)); err == nil {
		t.Fatal("auto-close notice send failure should remain retryable")
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
	for _, messageID := range []int64{2200, 2201} {
		if !fake.wasDeleted(messageID) {
			t.Fatalf("routine auto-close should delete message %d, deleted=%+v", messageID, fake.deleted)
		}
	}
	if fake.wasDeleted(2100) {
		t.Fatalf("routine card must stay as auto-close context, deleted=%+v", fake.deleted)
	}
	if closed.AutoCloseNoticeMessageID != nil || closed.AutoCloseNoticeSentAt != nil {
		t.Fatalf("failed notice delivery must remain retryable: %+v", closed)
	}

	fake.sendError = nil
	if err := approutine.RunCheckinTransitions(ctx, store, fake, nil, time.Date(2026, 6, 30, 0, 1, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	cardEdit, found := fake.findEdit(2100)
	if !found || cardEdit.ReplyMarkup == nil || len(cardEdit.ReplyMarkup.InlineKeyboard) != 0 || !strings.Contains(cardEdit.Text, "❌ зарядка") || !strings.Contains(cardEdit.Text, "❌ йога") {
		t.Fatalf("routine card was not finalized in place: found=%v edit=%+v", found, cardEdit)
	}
	lastNotice := fake.sent[len(fake.sent)-1]
	for _, part := range []string{`<a href="tg://user?id=42">Игорь</a>, время вышло`, `<a href="https://t.me/c/888000445/2000?thread=30">Рутина</a> за 28.06 закрыта`, "Неотмеченные пункты засчитаны как невыполненные"} {
		if !strings.Contains(lastNotice.Text, part) {
			t.Fatalf("auto-close notice missing %q: %+v", part, lastNotice)
		}
	}
	if lastNotice.ReplyToMessageID != 2100 || lastNotice.ReplyMarkup == nil || lastNotice.DisableNotification {
		t.Fatalf("auto-close notice must ping as a reply to its routine card: %+v", lastNotice)
	}
	closedWithNotice, found, err := q.GetRoutineCheckin(ctx, checkin.ID)
	if err != nil || !found || closedWithNotice.AutoCloseNoticeMessageID == nil || *closedWithNotice.AutoCloseNoticeMessageID != 3002 || closedWithNotice.AutoCloseNoticeSentAt == nil {
		t.Fatalf("auto-close notice was not stored found=%v checkin=%+v err=%v", found, closedWithNotice, err)
	}
	if tableEdit, ok := fake.findEdit(introID); !ok || !strings.Contains(tableEdit.Text, "Лидерборд") {
		t.Fatalf("routine table refresh missing: found=%v edit=%+v", ok, tableEdit)
	}

	if err := approutine.CleanupExpiredNotices(ctx, store, fake, time.Date(2026, 6, 30, 12, 1, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if fake.wasDeleted(3002) {
		t.Fatalf("fresh auto-close notice should stay for about 24h, deleted=%+v", fake.deleted)
	}

	if err := approutine.CleanupExpiredNotices(ctx, store, fake, time.Date(2026, 7, 1, 0, 1, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if !fake.wasDeleted(3002) {
		t.Fatalf("expired auto-close notice should be deleted, deleted=%+v", fake.deleted)
	}
	cleaned, found, err := q.GetRoutineCheckin(ctx, checkin.ID)
	if err != nil || !found || cleaned.AutoCloseNoticeMessageID != nil {
		t.Fatalf("auto-close notice id should be cleared found=%v checkin=%+v err=%v", found, cleaned, err)
	}
	sentAfterCleanup := len(fake.sent)
	if err := approutine.RunCheckinTransitions(ctx, store, fake, nil, time.Date(2026, 7, 1, 0, 2, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if len(fake.sent) != sentAfterCleanup {
		t.Fatalf("expired delivered notice must not be resurrected: before=%d after=%d", sentAfterCleanup, len(fake.sent))
	}
}

func TestRoutineAutoCloseFallsBackToRoutineSourceWhenCardIsMissing(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	ctx := context.Background()
	q := store.Queries()

	workspace, err := q.GetOrCreateWorkspace(ctx, -100888000447, "Group", "UTC")
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
	participant, err := q.RegisterParticipant(ctx, workspace.ID, 42, "igor", "Игорь")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := q.UpsertRoutinePlan(ctx, workspace.ID, participant.ID, participant.UserID, []string{"зарядка"}, 5000, 30)
	if err != nil {
		t.Fatal(err)
	}
	checkin, err := q.GetOrCreateRoutineCheckin(ctx, plan, time.Date(2026, 6, 28, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if err := q.SetRoutineCheckinCardMessageID(ctx, checkin.ID, 5100, 30); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	if _, completed, err := q.AutoFailRoutineCheckin(ctx, checkin.ID, now); err != nil || !completed {
		t.Fatalf("prepare auto-failed checkin: completed=%v err=%v", completed, err)
	}

	fake := &fakeTelegram{
		nextMessageID: 6000,
		editErrors: map[int64]error{
			5100: errors.New("Bad Request: message to edit not found"),
		},
	}
	if err := approutine.RunCheckinTransitions(ctx, store, fake, nil, now); err != nil {
		t.Fatal(err)
	}
	if len(fake.sent) != 1 || fake.sent[0].ReplyToMessageID != 5000 {
		t.Fatalf("missing card should fall back to the routine source reply: %+v", fake.sent)
	}
	if !strings.Contains(fake.sent[0].Text, `<a href="https://t.me/c/888000447/5000?thread=30">Рутина</a>`) {
		t.Fatalf("fallback notice lost routine source link: %+v", fake.sent[0])
	}
	stored, found, err := q.GetRoutineCheckin(ctx, checkin.ID)
	if err != nil || !found || stored.AutoCloseNoticeMessageID == nil || *stored.AutoCloseNoticeMessageID != 6001 {
		t.Fatalf("fallback notice was not stored: found=%v checkin=%+v err=%v", found, stored, err)
	}
}

func TestRoutineAutoCloseNoticeClaimSkipsConcurrentDelivery(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	ctx := context.Background()
	q := store.Queries()

	workspace, err := q.GetOrCreateWorkspace(ctx, -100888000448, "Group", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	participant, err := q.RegisterParticipant(ctx, workspace.ID, 42, "igor", "Игорь")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := q.UpsertRoutinePlan(ctx, workspace.ID, participant.ID, participant.UserID, []string{"зарядка"}, 5000, 30)
	if err != nil {
		t.Fatal(err)
	}
	checkin, err := q.GetOrCreateRoutineCheckin(ctx, plan, time.Date(2026, 6, 28, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if err := q.SetRoutineCheckinCardMessageID(ctx, checkin.ID, 5100, 30); err != nil {
		t.Fatal(err)
	}
	if _, completed, err := q.AutoFailRoutineCheckin(ctx, checkin.ID, time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)); err != nil || !completed {
		t.Fatalf("prepare auto-failed checkin: completed=%v err=%v", completed, err)
	}

	claimed := make(chan struct{})
	release := make(chan struct{})
	firstDone := make(chan error, 1)
	go func() {
		firstDone <- store.InTx(ctx, func(q *postgres.Queries) error {
			_, found, err := q.ClaimPendingRoutineAutoCloseNoticeContext(ctx)
			if err != nil {
				return err
			}
			if !found {
				return errors.New("first transaction did not claim pending notice")
			}
			close(claimed)
			<-release
			return nil
		})
	}()
	<-claimed

	type claimResult struct {
		found bool
		err   error
	}
	secondDone := make(chan claimResult, 1)
	go func() {
		var found bool
		err := store.InTx(ctx, func(q *postgres.Queries) error {
			var err error
			_, found, err = q.ClaimPendingRoutineAutoCloseNoticeContext(ctx)
			return err
		})
		secondDone <- claimResult{found: found, err: err}
	}()

	select {
	case result := <-secondDone:
		if result.err != nil || result.found {
			close(release)
			t.Fatalf("concurrent transaction must skip locked notice: found=%v err=%v", result.found, result.err)
		}
	case <-time.After(2 * time.Second):
		close(release)
		t.Fatal("concurrent notice claim blocked instead of using SKIP LOCKED")
	}
	close(release)
	if err := <-firstDone; err != nil {
		t.Fatal(err)
	}

	if err := store.InTx(ctx, func(q *postgres.Queries) error {
		_, found, err := q.ClaimPendingRoutineAutoCloseNoticeContext(ctx)
		if err != nil {
			return err
		}
		if !found {
			return errors.New("notice did not become claimable after lock release")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestCleanupExpiredNoticesDeletesOldRoutineReminder(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	ctx := context.Background()
	q := store.Queries()

	workspace, err := q.GetOrCreateWorkspace(ctx, -100888000446, "Group", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	participant, err := q.RegisterParticipant(ctx, workspace.ID, 42, "igor", "Igor")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := q.UpsertRoutinePlan(ctx, workspace.ID, participant.ID, participant.UserID, []string{"зарядка"}, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	checkin, err := q.GetOrCreateRoutineCheckin(ctx, plan, time.Date(2026, 6, 28, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if err := q.SetRoutineCheckinReminderMessageID(ctx, checkin.ID, 4100, time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}

	fake := &fakeTelegram{}
	if err := approutine.CleanupExpiredNotices(ctx, store, fake, time.Date(2026, 6, 29, 23, 59, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if fake.wasDeleted(4100) {
		t.Fatalf("routine reminder should stay until it is older than 24h, deleted=%+v", fake.deleted)
	}
	if err := approutine.CleanupExpiredNotices(ctx, store, fake, time.Date(2026, 6, 30, 0, 1, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if !fake.wasDeleted(4100) {
		t.Fatalf("expired routine reminder should be deleted, deleted=%+v", fake.deleted)
	}
	cleaned, found, err := q.GetRoutineCheckin(ctx, checkin.ID)
	if err != nil || !found || cleaned.ReminderMessageID != nil {
		t.Fatalf("routine reminder id should be cleared found=%v checkin=%+v err=%v", found, cleaned, err)
	}
}

type fakeTelegram struct {
	nextMessageID int64
	sent          []telegram.SendMessageRequest
	edits         []telegram.EditMessageTextRequest
	deleted       []int64
	sendError     error
	editErrors    map[int64]error
}

func (f *fakeTelegram) PollUpdates(context.Context, int64, int) ([]telegram.Update, error) {
	return nil, nil
}
func (f *fakeTelegram) AnswerCallbackQuery(context.Context, telegram.AnswerCallbackQueryRequest) error {
	return nil
}
func (f *fakeTelegram) SendMessage(_ context.Context, request telegram.SendMessageRequest) (telegram.Message, error) {
	f.sent = append(f.sent, request)
	if f.sendError != nil {
		return telegram.Message{}, f.sendError
	}
	f.nextMessageID++
	return telegram.Message{MessageID: f.nextMessageID, MessageThreadID: request.MessageThreadID, Chat: telegram.Chat{ID: request.ChatID, Type: "supergroup"}}, nil
}
func (f *fakeTelegram) EditMessageText(_ context.Context, request telegram.EditMessageTextRequest) error {
	f.edits = append(f.edits, request)
	if f.editErrors != nil && f.editErrors[request.MessageID] != nil {
		return f.editErrors[request.MessageID]
	}
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
