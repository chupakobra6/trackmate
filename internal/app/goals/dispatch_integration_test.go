package goals_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	appgoals "github.com/igor/trackmate/internal/app/goals"
	apppending "github.com/igor/trackmate/internal/app/pending"
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
	if !fake.hasSentToThread(40, "Вопросы по целям") {
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

	finalAt := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	if err := appgoals.DispatchWeeklyReviews(ctx, store, fake, finalAt); err != nil {
		t.Fatal(err)
	}
	if err := appgoals.DispatchFinalReviews(ctx, store, fake, finalAt); err != nil {
		t.Fatal(err)
	}
	if !fake.hasSentToThread(40, "Итог периода") {
		t.Fatalf("final review was not sent to goals topic: %+v", fake.sent)
	}
	finalRequest, found := fake.findSent("Итог периода")
	if !found || finalRequest.ReplyToMessageID != sourceMessageID {
		t.Fatalf("final review should reply to source goals message %d: found=%v request=%+v", sourceMessageID, found, finalRequest)
	}
}

func TestFinalReviewFallsBackWhenSourceReplyTargetIsMissing(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	ctx := context.Background()
	q := store.Queries()

	workspace, err := q.GetOrCreateWorkspace(ctx, -100888000556, "Group", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertTopicBinding(ctx, workspace.ID, domain.TopicGoals, 40, "Цели"); err != nil {
		t.Fatal(err)
	}
	participant, err := q.RegisterParticipant(ctx, workspace.ID, 42, "igor", "Игорь")
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
	goalSet, err := q.UpsertSeasonalGoalSet(ctx, workspace.ID, participant.ID, participant.UserID, period, "1. Работа", &sourceMessageID, &sourceThreadID)
	if err != nil {
		t.Fatal(err)
	}
	fake := &fakeTelegram{
		nextMessageID:   3000,
		sendReplyErrors: map[int64]error{sourceMessageID: errors.New("Bad Request: reply message not found")},
	}

	if err := appgoals.DispatchFinalReviews(ctx, store, fake, time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if len(fake.sent) != 2 || fake.sent[0].ReplyToMessageID != sourceMessageID || fake.sent[1].ReplyToMessageID != 0 {
		t.Fatalf("missing source should retry once without reply: %+v", fake.sent)
	}
	review, err := q.GetOrCreateGoalFinalReview(ctx, goalSet.ID)
	if err != nil || review.PromptMessageID == nil || *review.PromptMessageID != 3001 {
		t.Fatalf("fallback prompt was not persisted: review=%+v err=%v", review, err)
	}
}

func TestWeeklyDeleteFailureDoesNotBlockSkipOrFinalReview(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	ctx := context.Background()
	q := store.Queries()
	workspace, err := q.GetOrCreateWorkspace(ctx, -100888000559, "Group", "UTC")
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
		EndsOn:   time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
	}
	goalSet, err := q.UpsertSeasonalGoalSet(ctx, workspace.ID, participant.ID, participant.UserID, period, "Результат", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	fake := &fakeTelegram{nextMessageID: 6000, deleteErrors: map[int64]error{6100: errors.New("message can't be deleted")}}
	requestedAt := time.Date(2026, 6, 28, 20, 0, 0, 0, time.UTC)
	review, err := q.GetOrCreateGoalWeeklyReview(ctx, goalSet.ID, time.Date(2026, 6, 22, 0, 0, 0, 0, time.UTC), requestedAt)
	if err != nil {
		t.Fatal(err)
	}
	if updated, err := q.SetGoalWeeklyReviewReminderPrompt(ctx, review.ID, 6100, 40, requestedAt.Add(domain.GoalReviewReminderDelay)); err != nil || !updated {
		t.Fatalf("set retry prompt updated=%v err=%v", updated, err)
	}
	if _, err := q.UpsertPendingInput(ctx, workspace.ID, participant.UserID, 40, domain.PendingGoalWeeklyReview, map[string]any{"review_id": review.ID}); err != nil {
		t.Fatal(err)
	}
	skipAt := requestedAt.Add(domain.GoalReviewSkipAfter)
	setGoalTestClock(t, q, skipAt)
	if err := appgoals.DispatchWeeklyReviews(ctx, store, fake, skipAt); err != nil {
		t.Fatal(err)
	}
	if _, found, err := q.GetOpenGoalWeeklyReview(ctx, goalSet.ID); err != nil || found {
		t.Fatalf("weekly review stayed open found=%v err=%v", found, err)
	}
	if edit, found := fake.findEdit(6100); !found || !strings.Contains(edit.Text, "проверка целей закрыта") {
		t.Fatalf("undeletable prompt was not made inert: found=%v edit=%+v", found, edit)
	}
	if err := appgoals.DispatchFinalReviews(ctx, store, fake, skipAt); err != nil {
		t.Fatal(err)
	}
	if got := fake.countSent("Итог периода"); got != 1 {
		t.Fatalf("final review was blocked after delete failure: %d", got)
	}
}

func TestWeeklyReviewRetriesOnceAndSkipsBeforeTelegramDeleteLimit(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	ctx := context.Background()
	q := store.Queries()

	workspace, err := q.GetOrCreateWorkspace(ctx, -100888000557, "Group", "UTC")
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
		EndsOn:   time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
	}
	goalSet, err := q.UpsertSeasonalGoalSet(ctx, workspace.ID, participant.ID, participant.UserID, period, "Результат: предложение о работе", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	fake := &fakeTelegram{nextMessageID: 4000}
	requestedAt := time.Date(2026, 6, 28, 20, 0, 0, 0, time.UTC)
	setGoalTestClock(t, q, requestedAt)
	if err := appgoals.DispatchWeeklyReviews(ctx, store, fake, requestedAt); err != nil {
		t.Fatal(err)
	}
	if got := fake.countSent("Вопросы по целям"); got != 1 {
		t.Fatalf("initial prompts=%d want=1 sent=%+v", got, fake.sent)
	}
	initialPromptID := fake.sentMessageIDs[0]

	beforeReminder := requestedAt.Add(domain.GoalReviewReminderDelay - time.Second)
	setGoalTestClock(t, q, beforeReminder)
	if err := appgoals.DispatchWeeklyReviews(ctx, store, fake, beforeReminder); err != nil {
		t.Fatal(err)
	}
	if got := fake.countSent("Вопросы по целям"); got != 1 {
		t.Fatalf("prompt repeated before 24h: %d", got)
	}

	reminderAt := requestedAt.Add(domain.GoalReviewReminderDelay)
	setGoalTestClock(t, q, reminderAt)
	if err := apppending.CleanupStaleInputs(ctx, store, fake, reminderAt); err != nil {
		t.Fatal(err)
	}
	if fake.wasDeleted(initialPromptID) {
		t.Fatal("generic cleanup must not own weekly review prompts")
	}
	if err := appgoals.DispatchWeeklyReviews(ctx, store, fake, reminderAt); err != nil {
		t.Fatal(err)
	}
	if got := fake.countSent("Вопросы по целям"); got != 2 {
		t.Fatalf("prompts after 24h=%d want=2 sent=%+v", got, fake.sent)
	}
	if !fake.wasDeleted(initialPromptID) {
		t.Fatalf("initial prompt %d was not removed before retry: %+v", initialPromptID, fake.deleted)
	}
	retryPromptID := fake.sentMessageIDs[1]
	review, found, err := q.GetOpenGoalWeeklyReview(ctx, goalSet.ID)
	if err != nil || !found || review.ReminderSentAt == nil || review.SkippedAt != nil {
		t.Fatalf("open reminded review found=%v review=%+v err=%v", found, review, err)
	}

	after48Hours := requestedAt.Add(48 * time.Hour)
	setGoalTestClock(t, q, after48Hours)
	if err := apppending.CleanupStaleInputs(ctx, store, fake, after48Hours); err != nil {
		t.Fatal(err)
	}
	if fake.wasDeleted(retryPromptID) {
		t.Fatal("retry prompt must remain answerable before its close deadline")
	}
	if pending, found, err := q.GetPendingInput(ctx, workspace.ID, participant.UserID, 40); err != nil || !found || pending.Kind != domain.PendingGoalWeeklyReview {
		t.Fatalf("retry pending must survive 48h found=%v pending=%+v err=%v", found, pending, err)
	}
	if err := appgoals.DispatchWeeklyReviews(ctx, store, fake, after48Hours); err != nil {
		t.Fatal(err)
	}
	if got := fake.countSent("Вопросы по целям"); got != 2 {
		t.Fatalf("second retry was sent: %d", got)
	}

	beforeSkip := requestedAt.Add(domain.GoalReviewSkipAfter - time.Second)
	setGoalTestClock(t, q, beforeSkip)
	if err := appgoals.DispatchFinalReviews(ctx, store, fake, beforeSkip); err != nil {
		t.Fatal(err)
	}
	if got := fake.countSent("Итог периода"); got != 0 {
		t.Fatalf("final review overlapped open weekly review: %d", got)
	}

	skipAt := requestedAt.Add(domain.GoalReviewSkipAfter)
	setGoalTestClock(t, q, skipAt)
	if err := appgoals.DispatchWeeklyReviews(ctx, store, fake, skipAt); err != nil {
		t.Fatal(err)
	}
	if !fake.wasDeleted(retryPromptID) {
		t.Fatalf("retry prompt %d was not removed at the safe deadline: %+v", retryPromptID, fake.deleted)
	}
	if _, found, err := q.GetPendingInput(ctx, workspace.ID, participant.UserID, 40); err != nil || found {
		t.Fatalf("weekly pending must be closed at deadline found=%v err=%v", found, err)
	}
	if _, found, err := q.GetOpenGoalWeeklyReview(ctx, goalSet.ID); err != nil || found {
		t.Fatalf("weekly review must be persisted as skipped found=%v err=%v", found, err)
	}
	var skippedAt time.Time
	if err := store.Pool().QueryRow(ctx, `SELECT skipped_at FROM seasonal_goal_weekly_reviews WHERE goal_set_id = $1`, goalSet.ID).Scan(&skippedAt); err != nil {
		t.Fatal(err)
	}
	if !skippedAt.Equal(skipAt) {
		t.Fatalf("skipped_at=%s want=%s", skippedAt, skipAt)
	}
	if _, saved, err := q.SubmitGoalWeeklyReview(ctx, review.ID, participant.UserID, "late answer", 999, 40); err != nil || saved {
		t.Fatalf("skipped review accepted a late answer saved=%v err=%v", saved, err)
	}
	if err := appgoals.DispatchFinalReviews(ctx, store, fake, skipAt); err != nil {
		t.Fatal(err)
	}
	if got := fake.countSent("Итог периода"); got != 1 {
		t.Fatalf("final review was not released after weekly skip: %d", got)
	}
	if err := appgoals.DispatchWeeklyReviews(ctx, store, fake, skipAt.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if got := fake.countSent("Вопросы по целям"); got != 2 {
		t.Fatalf("skipped review was sent again: %d", got)
	}
}

func TestWeeklyReviewDoesNotReplaceAnotherGoalsPending(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	ctx := context.Background()
	q := store.Queries()

	workspace, err := q.GetOrCreateWorkspace(ctx, -100888000558, "Group", "UTC")
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
	if _, err := q.UpsertSeasonalGoalSet(ctx, workspace.ID, participant.ID, participant.UserID, period, "Результат: предложение о работе", nil, nil); err != nil {
		t.Fatal(err)
	}

	fake := &fakeTelegram{nextMessageID: 5000}
	requestedAt := time.Date(2026, 6, 28, 20, 0, 0, 0, time.UTC)
	setGoalTestClock(t, q, requestedAt)
	if err := appgoals.DispatchWeeklyReviews(ctx, store, fake, requestedAt); err != nil {
		t.Fatal(err)
	}
	initialPromptID := fake.sentMessageIDs[0]
	if _, err := q.UpsertPendingInput(ctx, workspace.ID, participant.UserID, 40, domain.PendingSeasonalGoals, map[string]any{"prompt_message_id": 9001}); err != nil {
		t.Fatal(err)
	}

	reminderAt := requestedAt.Add(domain.GoalReviewReminderDelay)
	setGoalTestClock(t, q, reminderAt)
	if err := appgoals.DispatchWeeklyReviews(ctx, store, fake, reminderAt); err != nil {
		t.Fatal(err)
	}
	if got := fake.countSent("Вопросы по целям"); got != 1 || fake.wasDeleted(initialPromptID) {
		t.Fatalf("weekly review replaced another pending sent=%d deleted=%+v", got, fake.deleted)
	}

	skipAt := requestedAt.Add(domain.GoalReviewSkipAfter)
	setGoalTestClock(t, q, skipAt)
	if err := appgoals.DispatchWeeklyReviews(ctx, store, fake, skipAt); err != nil {
		t.Fatal(err)
	}
	if pending, found, err := q.GetPendingInput(ctx, workspace.ID, participant.UserID, 40); err != nil || !found || pending.Kind != domain.PendingSeasonalGoals {
		t.Fatalf("another pending was changed found=%v pending=%+v err=%v", found, pending, err)
	}
	if !fake.wasDeleted(initialPromptID) {
		t.Fatalf("expired weekly prompt was not removed: %+v", fake.deleted)
	}
}

func setGoalTestClock(t *testing.T, q interface {
	SetClockOverride(context.Context, *time.Time) error
}, now time.Time) {
	t.Helper()
	if err := q.SetClockOverride(context.Background(), &now); err != nil {
		t.Fatal(err)
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
	nextMessageID   int64
	sent            []telegram.SendMessageRequest
	sentMessageIDs  []int64
	deleted         []int64
	edits           []telegram.EditMessageTextRequest
	deleteErrors    map[int64]error
	sendReplyErrors map[int64]error
}

func (f *fakeTelegram) PollUpdates(context.Context, int64, int) ([]telegram.Update, error) {
	return nil, nil
}
func (f *fakeTelegram) AnswerCallbackQuery(context.Context, telegram.AnswerCallbackQueryRequest) error {
	return nil
}
func (f *fakeTelegram) SendMessage(_ context.Context, request telegram.SendMessageRequest) (telegram.Message, error) {
	f.sent = append(f.sent, request)
	if err := f.sendReplyErrors[request.ReplyToMessageID]; err != nil {
		return telegram.Message{}, err
	}
	f.nextMessageID++
	f.sentMessageIDs = append(f.sentMessageIDs, f.nextMessageID)
	return telegram.Message{MessageID: f.nextMessageID, MessageThreadID: request.MessageThreadID, Chat: telegram.Chat{ID: request.ChatID, Type: "supergroup"}}, nil
}
func (f *fakeTelegram) EditMessageText(_ context.Context, request telegram.EditMessageTextRequest) error {
	f.edits = append(f.edits, request)
	return nil
}
func (f *fakeTelegram) DeleteMessage(_ context.Context, _ int64, messageID int64) error {
	f.deleted = append(f.deleted, messageID)
	if err := f.deleteErrors[messageID]; err != nil {
		return err
	}
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

func (f *fakeTelegram) countSent(text string) int {
	count := 0
	for _, sent := range f.sent {
		if strings.Contains(sent.Text, text) {
			count++
		}
	}
	return count
}

func (f *fakeTelegram) findSent(text string) (telegram.SendMessageRequest, bool) {
	for _, sent := range f.sent {
		if strings.Contains(sent.Text, text) {
			return sent, true
		}
	}
	return telegram.SendMessageRequest{}, false
}

func (f *fakeTelegram) wasDeleted(messageID int64) bool {
	for _, deleted := range f.deleted {
		if deleted == messageID {
			return true
		}
	}
	return false
}

func (f *fakeTelegram) findEdit(messageID int64) (telegram.EditMessageTextRequest, bool) {
	for i := len(f.edits) - 1; i >= 0; i-- {
		if f.edits[i].MessageID == messageID {
			return f.edits[i], true
		}
	}
	return telegram.EditMessageTextRequest{}, false
}
