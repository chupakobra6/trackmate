package pending

import (
	"context"
	"encoding/json"
	"time"

	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/storage/postgres"
	"github.com/igor/trackmate/internal/telegram"
)

const staleBatchSize = 50

func CleanupStaleInputs(ctx context.Context, store *postgres.Store, tg telegram.API, nowUTC time.Time) error {
	cutoff := nowUTC.UTC().Add(-domain.PendingInputMaxAge)
	for {
		items, err := store.Queries().ListStalePendingInputContexts(ctx, cutoff, staleBatchSize)
		if err != nil || len(items) == 0 {
			return err
		}
		for _, item := range items {
			claimed, ok, err := store.Queries().ClaimStalePendingInput(ctx, item.Pending.ID, cutoff)
			if err != nil {
				return err
			}
			if !ok {
				continue
			}
			deletePendingMessages(ctx, tg, item.Workspace.ChatID, claimed.Payload)
		}
		if len(items) < staleBatchSize {
			return nil
		}
	}
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
