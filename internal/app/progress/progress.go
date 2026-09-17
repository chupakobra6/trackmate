package progress

import (
	"context"
	"errors"
	"time"

	"github.com/igor/trackmate/internal/app/delivery"
	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/storage/postgres"
	"github.com/igor/trackmate/internal/telegram"
	"github.com/igor/trackmate/internal/ui"
)

func PublishPending(ctx context.Context, store *postgres.Store, tg telegram.API) error {
	now, err := store.Queries().CurrentNow(ctx, time.Now().UTC())
	if err != nil {
		return err
	}
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
			retryErr := delivery.Defer(ctx, store.Queries(), "progress", event.ID, now, err)
			requeueErr := store.Queries().RequeueProgressEvent(ctx, event.ID)
			if retryErr != nil || requeueErr != nil {
				return errors.Join(retryErr, requeueErr)
			}
			continue
		}
		if err := store.Queries().MarkProgressEventPublished(ctx, event.ID, message.MessageID, message.Date()); err != nil {
			compensateErr := delivery.CompensateSentMessage(tg, workspace.ChatID, message.MessageID, func(cleanupCtx context.Context) error {
				return store.Queries().RequeueProgressEvent(cleanupCtx, event.ID)
			})
			return errors.Join(err, compensateErr)
		}
		if err := store.Queries().ClearDeliveryRetry(ctx, "progress", event.ID); err != nil {
			return err
		}
	}
}
