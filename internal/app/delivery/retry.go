package delivery

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/igor/trackmate/internal/storage/postgres"
	"github.com/igor/trackmate/internal/telegram"
)

// Run retries a single delivery, while its domain owner remains responsible for
// recording success and compensating sends whose confirmation could not persist.
func Run(ctx context.Context, q *postgres.Queries, operation string, id int64, now time.Time, fn func() error) error {
	ready, err := q.DeliveryReady(ctx, operation, id, now)
	if err != nil || !ready {
		return err
	}
	if err = fn(); err != nil {
		return Defer(ctx, q, operation, id, now, err)
	}
	return q.ClearDeliveryRetry(ctx, operation, id)
}

// Defer records Telegram failures only. Storage, cancellation and unknown errors
// remain fatal; a service-wide Telegram failure is both recorded and surfaced.
func Defer(ctx context.Context, q *postgres.Queries, operation string, id int64, now time.Time, cause error) error {
	var apiErr *telegram.Error
	if !onlyTelegramErrors(cause) || !errors.As(cause, &apiErr) {
		return cause
	}
	if ctx.Err() != nil {
		if !apiErr.Ambiguous {
			return errors.Join(cause, ctx.Err())
		}
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		ctx = cleanupCtx
	}
	if err := q.RecordDeliveryFailure(ctx, operation, id, now, apiErr.Error(), apiErr.Ambiguous); err != nil {
		return errors.Join(cause, err)
	}
	if apiErr.Ambiguous {
		return fmt.Errorf("delivery outcome unknown; reconcile %s %d before retry: %w", operation, id, cause)
	}
	slog.WarnContext(ctx, "worker_delivery_deferred", "operation", operation, "entity_id", id, "error", apiErr)
	if apiErr.StatusCode == 401 || apiErr.StatusCode == 429 || apiErr.StatusCode >= 500 || apiErr.StatusCode == 0 {
		return fmt.Errorf("Telegram unavailable during %s: %w", operation, cause)
	}
	return nil
}

func onlyTelegramErrors(err error) bool {
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		for _, part := range joined.Unwrap() {
			if !onlyTelegramErrors(part) {
				return false
			}
		}
		return true
	}
	if wrapped, ok := err.(interface{ Unwrap() error }); ok {
		return onlyTelegramErrors(wrapped.Unwrap())
	}
	_, ok := err.(*telegram.Error)
	return ok
}
