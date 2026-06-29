package routine

import (
	"context"
	"encoding/json"
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

func RunCheckinTransitions(ctx context.Context, store *postgres.Store, tg telegram.API, logger *slog.Logger, nowUTC time.Time) error {
	checkins, err := store.Queries().ListOpenRoutineCheckinContexts(ctx)
	if err != nil {
		return err
	}
	refreshed := map[int64]postgres.Workspace{}
	for _, item := range checkins {
		if item.Checkin.CardMessageID == nil || item.Checkin.CardMessageThreadID == nil {
			continue
		}
		autoFailDue, err := domain.RoutineAutoFailDue(item.Checkin.CheckinDate, item.Workspace.Timezone, item.Checkin.CompletedAt, nowUTC)
		if err != nil {
			return err
		}
		if autoFailDue {
			pending, pendingFound, err := store.Queries().GetPendingInput(ctx, item.Workspace.ID, item.Participant.UserID, *item.Checkin.CardMessageThreadID)
			if err != nil {
				return err
			}
			if pendingFound && pending.Kind == domain.PendingRoutineReason && payloadInt64(pending.Payload, "checkin_id") == item.Checkin.ID {
				deletePendingMessages(ctx, tg, item.Workspace.ChatID, pending.Payload)
			}
			if err := store.Queries().ClearRoutineCheckinPendingInput(ctx, item.Workspace.ID, item.Participant.UserID, *item.Checkin.CardMessageThreadID, item.Checkin.ID); err != nil {
				return err
			}
			closed, completed, err := store.Queries().AutoFailRoutineCheckin(ctx, item.Checkin.ID, nowUTC)
			if err != nil {
				return err
			}
			if !completed {
				continue
			}
			if item.Checkin.ReminderMessageID != nil {
				_ = tg.DeleteMessage(ctx, item.Workspace.ChatID, *item.Checkin.ReminderMessageID)
				if err := store.Queries().ClearRoutineCheckinReminderMessageID(ctx, item.Checkin.ID); err != nil {
					return err
				}
			}
			_ = tg.DeleteMessage(ctx, item.Workspace.ChatID, *item.Checkin.CardMessageID)
			notice, err := tg.SendMessage(ctx, telegram.SendMessageRequest{
				ChatID:              item.Workspace.ChatID,
				MessageThreadID:     *item.Checkin.CardMessageThreadID,
				Text:                ui.RoutineAutoClosedText(closed),
				ReplyMarkup:         ui.DismissKeyboard(),
				DisableNotification: true,
			})
			if err != nil {
				if logger != nil {
					logger.WarnContext(ctx, "routine_auto_close_notice_failed", "checkin_id", item.Checkin.ID, "error", err)
				}
				return err
			}
			if err := store.Queries().SetRoutineCheckinAutoCloseNoticeMessageID(ctx, item.Checkin.ID, notice.MessageID, nowUTC); err != nil {
				return err
			}
			refreshed[item.Workspace.ID] = item.Workspace
			continue
		}
		reminderDue, err := domain.RoutineReminderDue(item.Checkin.CheckinDate, item.Workspace.Timezone, item.Checkin.ReminderSentAt, item.Checkin.CompletedAt, nowUTC)
		if err != nil {
			return err
		}
		if !reminderDue {
			continue
		}
		message, err := tg.SendMessage(ctx, telegram.SendMessageRequest{
			ChatID:              item.Workspace.ChatID,
			MessageThreadID:     *item.Checkin.CardMessageThreadID,
			Text:                ui.RoutineReminderText(item.Checkin),
			ReplyMarkup:         ui.DismissKeyboard(),
			ReplyToMessageID:    *item.Checkin.CardMessageID,
			DisableNotification: true,
		})
		if err != nil {
			if logger != nil {
				logger.WarnContext(ctx, "routine_reminder_dispatch_failed", "checkin_id", item.Checkin.ID, "error", err)
			}
			return err
		}
		if err := store.Queries().SetRoutineCheckinReminderMessageID(ctx, item.Checkin.ID, message.MessageID, nowUTC); err != nil {
			return err
		}
	}
	for _, workspace := range refreshed {
		if err := RefreshLeaderboard(ctx, store.Queries(), tg, workspace, workspace.ChatID, nowUTC); err != nil {
			return err
		}
	}
	return nil
}

func CleanupExpiredNotices(ctx context.Context, store *postgres.Store, tg telegram.API, nowUTC time.Time) error {
	cutoff := nowUTC.UTC().Add(-domain.RoutineNoticeMaxAge)
	notices, err := store.Queries().ListExpiredRoutineNoticeContexts(ctx, cutoff)
	if err != nil {
		return err
	}
	for _, notice := range notices {
		if notice.Checkin.ReminderMessageID != nil && notice.Checkin.ReminderSentAt != nil && !notice.Checkin.ReminderSentAt.After(cutoff) {
			_ = tg.DeleteMessage(ctx, notice.Workspace.ChatID, *notice.Checkin.ReminderMessageID)
			if err := store.Queries().ClearRoutineCheckinReminderMessageID(ctx, notice.Checkin.ID); err != nil {
				return err
			}
		}
		if notice.Checkin.AutoCloseNoticeMessageID != nil && notice.Checkin.AutoCloseNoticeSentAt != nil && !notice.Checkin.AutoCloseNoticeSentAt.After(cutoff) {
			_ = tg.DeleteMessage(ctx, notice.Workspace.ChatID, *notice.Checkin.AutoCloseNoticeMessageID)
			if err := store.Queries().ClearRoutineCheckinAutoCloseNoticeMessageID(ctx, notice.Checkin.ID); err != nil {
				return err
			}
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

func deletePendingMessages(ctx context.Context, tg telegram.API, chatID int64, payload map[string]any) {
	seen := map[int64]bool{}
	for _, messageID := range append([]int64{payloadInt64(payload, "prompt_message_id")}, payloadInt64Slice(payload, "user_message_ids")...) {
		if messageID == 0 || seen[messageID] {
			continue
		}
		seen[messageID] = true
		_ = tg.DeleteMessage(ctx, chatID, messageID)
	}
}

func payloadInt64(payload map[string]any, key string) int64 {
	switch value := payload[key].(type) {
	case float64:
		return int64(value)
	case int64:
		return value
	case int:
		return int64(value)
	case json.Number:
		result, _ := value.Int64()
		return result
	default:
		return 0
	}
}

func payloadInt64Slice(payload map[string]any, key string) []int64 {
	switch values := payload[key].(type) {
	case []int64:
		return values
	case []any:
		result := make([]int64, 0, len(values))
		for _, value := range values {
			switch typed := value.(type) {
			case float64:
				result = append(result, int64(typed))
			case int64:
				result = append(result, typed)
			case int:
				result = append(result, int64(typed))
			case json.Number:
				parsed, _ := typed.Int64()
				result = append(result, parsed)
			}
		}
		return result
	default:
		return nil
	}
}
