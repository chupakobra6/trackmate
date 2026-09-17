package delivery_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/igor/trackmate/internal/app/delivery"
	"github.com/igor/trackmate/internal/telegram"
	"github.com/igor/trackmate/internal/testsupport"
)

func TestRetryDoesNotHidePersistenceFailureJoinedWithCompensationError(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	persistence := errors.New("database write failed")
	err := delivery.Run(context.Background(), store.Queries(), "routine_card", 1, time.Now(), func() error {
		return errors.Join(persistence, &telegram.Error{StatusCode: 400, Description: "compensation failed"})
	})
	if !errors.Is(err, persistence) {
		t.Fatalf("persistence failure hidden: %v", err)
	}
	var count int
	if err := store.Pool().QueryRow(context.Background(), `SELECT count(*) FROM worker_delivery_retries`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("incorrect automatic retry count=%d err=%v", count, err)
	}
}
