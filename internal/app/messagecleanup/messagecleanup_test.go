package messagecleanup

import (
	"context"
	"testing"
	"time"

	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/telegram"
)

func TestCloseBestEffortHonorsTelegramDeletionWindow(t *testing.T) {
	sentAt := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	fallback := telegram.EditMessageTextRequest{Text: "closed"}

	tests := []struct {
		name        string
		now         time.Time
		deleteError error
		wantDelete  bool
		wantEdit    bool
	}{
		{name: "inside window", now: sentAt.Add(domain.TelegramDeleteTargetAge), wantDelete: true},
		{name: "delete failure", now: sentAt.Add(domain.TelegramDeleteTargetAge), deleteError: context.DeadlineExceeded, wantDelete: true, wantEdit: true},
		{name: "at telegram limit", now: sentAt.Add(domain.TelegramDeleteLimit), wantEdit: true},
		{name: "after telegram limit", now: sentAt.Add(domain.TelegramDeleteLimit + time.Hour), wantEdit: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeAPI{deleteError: tt.deleteError}
			CloseBestEffort(context.Background(), fake, -1001, 42, sentAt, tt.now, fallback)
			if got := len(fake.deleted) > 0; got != tt.wantDelete {
				t.Fatalf("delete called=%v want=%v", got, tt.wantDelete)
			}
			if got := len(fake.edited) > 0; got != tt.wantEdit {
				t.Fatalf("edit called=%v want=%v", got, tt.wantEdit)
			}
		})
	}
}

func TestDeleteBestEffortSkipsUnknownOrExpiredAge(t *testing.T) {
	sentAt := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	fake := &fakeAPI{}
	DeleteBestEffort(context.Background(), fake, -1001, 41, sentAt, sentAt.Add(domain.TelegramDeleteTargetAge))
	DeleteBestEffort(context.Background(), fake, -1001, 42, sentAt, sentAt.Add(domain.TelegramDeleteLimit))
	DeleteBestEffort(context.Background(), fake, -1001, 43, time.Time{}, sentAt)
	if len(fake.deleted) != 1 || fake.deleted[0] != 41 {
		t.Fatalf("deleted=%v want=[41]", fake.deleted)
	}
}

type fakeAPI struct {
	deleted     []int64
	edited      []telegram.EditMessageTextRequest
	deleteError error
}

func (f *fakeAPI) DeleteMessage(_ context.Context, _ int64, messageID int64) error {
	f.deleted = append(f.deleted, messageID)
	return f.deleteError
}

func (f *fakeAPI) EditMessageText(_ context.Context, request telegram.EditMessageTextRequest) error {
	f.edited = append(f.edited, request)
	return nil
}
