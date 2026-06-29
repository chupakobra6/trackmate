package progress

import (
	"context"

	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/storage/postgres"
	"github.com/igor/trackmate/internal/telegram"
	"github.com/igor/trackmate/internal/ui"
)

func PublishPending(ctx context.Context, store *postgres.Store, tg telegram.API) error {
	for {
		event, ok, err := store.Queries().ClaimProgressEvent(ctx)
		if err != nil || !ok {
			return err
		}
		workspace, found, err := store.Queries().GetWorkspaceByID(ctx, event.WorkspaceGroupID)
		if err != nil {
			_ = store.Queries().RequeueProgressEvent(ctx, event.ID)
			return err
		}
		if !found {
			_ = store.Queries().MarkProgressEventFailed(ctx, event.ID)
			continue
		}
		progressTopic, found, err := store.Queries().GetTopicBinding(ctx, workspace.ID, domain.TopicProgress)
		if err != nil {
			_ = store.Queries().RequeueProgressEvent(ctx, event.ID)
			return err
		}
		if !found {
			_ = store.Queries().MarkProgressEventFailed(ctx, event.ID)
			continue
		}
		disablePreview := true
		message, err := tg.SendMessage(ctx, telegram.SilentMessage(telegram.SendMessageRequest{
			ChatID:                workspace.ChatID,
			MessageThreadID:       progressTopic.ThreadID,
			Text:                  ui.FormatProgressEvent(event),
			DisableWebPagePreview: &disablePreview,
		}))
		if err != nil {
			if telegram.IsTransientRequestError(err) {
				_ = store.Queries().RequeueProgressEvent(ctx, event.ID)
			} else {
				_ = store.Queries().MarkProgressEventFailed(ctx, event.ID)
			}
			continue
		}
		if err := store.Queries().MarkProgressEventPublished(ctx, event.ID, message.MessageID, message.Date()); err != nil {
			return err
		}
	}
}
