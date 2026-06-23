package routine

import (
	"context"
	"log/slog"
	"time"

	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/storage/postgres"
	"github.com/igor/trackmate/internal/telegram"
	"github.com/igor/trackmate/internal/ui"
)

func DispatchDueCheckins(ctx context.Context, store *postgres.Store, tg telegram.API, logger *slog.Logger, nowUTC time.Time) error {
	plans, err := store.Queries().ListRoutinePlanContexts(ctx)
	if err != nil {
		return err
	}
	for _, item := range plans {
		checkinDate, due, err := domain.RoutineCheckinDue(item.Plan.CreatedAt, item.Workspace.Timezone, nowUTC)
		if err != nil {
			return err
		}
		if !due || len(item.Plan.Items) == 0 {
			continue
		}
		routineTopic, found, err := store.Queries().GetTopicBinding(ctx, item.Workspace.ID, domain.TopicRoutine)
		if err != nil {
			return err
		}
		if !found {
			continue
		}
		checkin, err := store.Queries().GetOrCreateRoutineCheckin(ctx, item.Plan, checkinDate)
		if err != nil {
			return err
		}
		if checkin.CompletedAt != nil || checkin.CardMessageID != nil {
			continue
		}
		nextIndex := ui.NextRoutineItemIndex(checkin)
		if nextIndex < 0 {
			continue
		}
		message, err := tg.SendMessage(ctx, telegram.SendMessageRequest{
			ChatID:              item.Workspace.ChatID,
			MessageThreadID:     routineTopic.ThreadID,
			Text:                ui.FormatRoutineCheckinCard(checkin, item.Participant.DisplayName, participantUsername(item.Participant), ""),
			ReplyMarkup:         ui.RoutineItemKeyboard(checkin.ID, nextIndex),
			DisableNotification: true,
		})
		if err != nil {
			if logger != nil {
				logger.WarnContext(ctx, "routine_checkin_dispatch_failed", "checkin_id", checkin.ID, "error", err)
			}
			return err
		}
		if err := store.Queries().SetRoutineCheckinCardMessageID(ctx, checkin.ID, message.MessageID, routineTopic.ThreadID); err != nil {
			return err
		}
	}
	return nil
}

func RefreshLeaderboard(ctx context.Context, q *postgres.Queries, tg telegram.API, workspace postgres.Workspace, chatID int64, nowUTC time.Time) error {
	binding, found, err := q.GetTopicBinding(ctx, workspace.ID, domain.TopicRoutine)
	if err != nil || !found {
		return err
	}
	entries, err := q.GetRoutineLeaderboard(ctx, workspace.ID, nowUTC)
	if err != nil {
		return err
	}
	text := ui.FormatRoutineLeaderboard(entries)
	if binding.IntroMessageID != nil {
		if err := tg.EditMessageText(ctx, telegram.EditMessageTextRequest{ChatID: chatID, MessageID: *binding.IntroMessageID, Text: text}); err == nil {
			return nil
		}
	}
	message, err := tg.SendMessage(ctx, telegram.SendMessageRequest{
		ChatID:              chatID,
		MessageThreadID:     binding.ThreadID,
		Text:                text,
		DisableNotification: true,
	})
	if err != nil {
		return err
	}
	return q.SetTopicMessages(ctx, workspace.ID, domain.TopicRoutine, &message.MessageID, nil, false, false)
}

func participantUsername(participant postgres.Participant) string {
	if participant.Username == nil {
		return ""
	}
	return *participant.Username
}
