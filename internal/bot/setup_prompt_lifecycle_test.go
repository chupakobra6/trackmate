package bot_test

import (
	"context"
	"errors"
	"testing"

	"github.com/igor/trackmate/internal/bot"
	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/logging"
	"github.com/igor/trackmate/internal/storage/postgres"
	"github.com/igor/trackmate/internal/telegram"
	"github.com/igor/trackmate/internal/testsupport"
)

func TestSetupConfigureReusesExistingPromptWithoutSendingDuplicate(t *testing.T) {
	for _, fixture := range setupPromptFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			store, _ := testsupport.OpenMigratedStore(t)
			ctx := context.Background()
			workspace := createSetupPromptWorkspace(t, ctx, store, fixture.topic, fixture.threadID)
			fake := newFakeTelegram()
			service := bot.NewService(store, fake, logging.New("ERROR"), "UTC", 99)
			update := setupPromptUpdate(workspace.ChatID, fixture.threadID, fixture.callbackData)

			if _, err := service.HandleUpdate(ctx, update); err != nil {
				t.Fatal(err)
			}
			answer, err := service.HandleUpdate(ctx, update)
			if err != nil {
				t.Fatal(err)
			}
			if answer.Text == "" {
				t.Fatal("repeated callback should explain that the input is already open")
			}
			if len(fake.sent) != 1 {
				t.Fatalf("repeated callback sent a duplicate prompt: %+v", fake.sent)
			}
			if len(fake.edits) != 1 || fake.edits[0].MessageID != 1001 {
				t.Fatalf("stored prompt was not verified in place: %+v", fake.edits)
			}
		})
	}
}

func TestSetupConfigureReplacesOnlyConfirmedMissingPrompt(t *testing.T) {
	for _, fixture := range setupPromptFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			store, _ := testsupport.OpenMigratedStore(t)
			ctx := context.Background()
			workspace := createSetupPromptWorkspace(t, ctx, store, fixture.topic, fixture.threadID)
			fake := newFakeTelegram()
			service := bot.NewService(store, fake, logging.New("ERROR"), "UTC", 99)
			update := setupPromptUpdate(workspace.ChatID, fixture.threadID, fixture.callbackData)

			if _, err := service.HandleUpdate(ctx, update); err != nil {
				t.Fatal(err)
			}
			fake.editErrors = map[int64]error{1001: errors.New("Bad Request: message to edit not found")}
			if _, err := service.HandleUpdate(ctx, update); err != nil {
				t.Fatal(err)
			}
			if len(fake.sent) != 2 {
				t.Fatalf("missing prompt should produce one replacement: %+v", fake.sent)
			}
			pending, found, err := store.Queries().GetPendingInput(ctx, workspace.ID, 42, fixture.threadID)
			if err != nil || !found || pending.Kind != fixture.kind || pending.Payload["prompt_message_id"] != float64(1002) {
				t.Fatalf("pending did not move to replacement prompt: found=%v pending=%+v err=%v", found, pending, err)
			}
		})
	}
}

func TestSetupConfigureDoesNotSendReplacementOnTransientEditFailure(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	ctx := context.Background()
	fixture := setupPromptFixtures()[0]
	workspace := createSetupPromptWorkspace(t, ctx, store, fixture.topic, fixture.threadID)
	fake := newFakeTelegram()
	service := bot.NewService(store, fake, logging.New("ERROR"), "UTC", 99)
	update := setupPromptUpdate(workspace.ChatID, fixture.threadID, fixture.callbackData)

	if _, err := service.HandleUpdate(ctx, update); err != nil {
		t.Fatal(err)
	}
	fake.editErrors = map[int64]error{1001: errors.New("request timeout")}
	if _, err := service.HandleUpdate(ctx, update); err == nil {
		t.Fatal("transient edit failure should be returned instead of creating a duplicate")
	}
	if len(fake.sent) != 1 {
		t.Fatalf("transient failure sent a replacement prompt: %+v", fake.sent)
	}
	pending, found, err := store.Queries().GetPendingInput(ctx, workspace.ID, 42, fixture.threadID)
	if err != nil || !found || pending.Payload["prompt_message_id"] != float64(1001) {
		t.Fatalf("transient failure changed pending prompt: found=%v pending=%+v err=%v", found, pending, err)
	}
}

type setupPromptFixture struct {
	name         string
	topic        domain.TopicKey
	threadID     int64
	callbackData string
	kind         domain.PendingInputKind
}

func setupPromptFixtures() []setupPromptFixture {
	return []setupPromptFixture{
		{name: "goals", topic: domain.TopicGoals, threadID: 14, callbackData: "goals:configure", kind: domain.PendingSeasonalGoals},
		{name: "routine", topic: domain.TopicRoutine, threadID: 13, callbackData: "routine:configure", kind: domain.PendingRoutinePlan},
	}
}

func createSetupPromptWorkspace(t *testing.T, ctx context.Context, store *postgres.Store, topic domain.TopicKey, threadID int64) postgres.Workspace {
	t.Helper()
	workspace, err := store.Queries().GetOrCreateWorkspace(ctx, -1001234567890, "Group", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Queries().UpsertTopicBinding(ctx, workspace.ID, topic, threadID, string(topic)); err != nil {
		t.Fatal(err)
	}
	return workspace
}

func setupPromptUpdate(chatID int64, threadID int64, data string) telegram.Update {
	return telegram.Update{Callback: &telegram.CallbackQuery{
		ID:   data,
		From: telegram.User{ID: 42, Username: "igor", FirstName: "Игорь"},
		Data: data,
		Message: &telegram.Message{
			MessageID:       200,
			MessageThreadID: threadID,
			Chat:            telegram.Chat{ID: chatID, Type: "supergroup", Title: "Group", IsForum: true},
		},
	}}
}
