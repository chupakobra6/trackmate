package postgres

import (
	"testing"
	"time"

	"github.com/igor/trackmate/internal/domain"
)

func TestRoutineStreaksTreatPartialAsNeutral(t *testing.T) {
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	done := domain.RoutineItemDone
	partial := domain.RoutineItemPartial
	failed := domain.RoutineItemFailed

	tests := []struct {
		name        string
		checkins    []RoutineCheckin
		wantCurrent int
		wantMax     int
	}{
		{
			name: "partial preserves but does not increase the series",
			checkins: []RoutineCheckin{
				streakCheckin(start, &done),
				streakCheckin(start.AddDate(0, 0, 1), &partial),
				streakCheckin(start.AddDate(0, 0, 2), &done),
			},
			wantCurrent: 2,
			wantMax:     2,
		},
		{
			name: "partial after full days keeps the existing number",
			checkins: []RoutineCheckin{
				streakCheckin(start, &done),
				streakCheckin(start.AddDate(0, 0, 1), &done),
				streakCheckin(start.AddDate(0, 0, 2), &partial),
			},
			wantCurrent: 2,
			wantMax:     2,
		},
		{
			name: "failed resets the current series",
			checkins: []RoutineCheckin{
				streakCheckin(start, &done),
				streakCheckin(start.AddDate(0, 0, 1), &failed),
			},
			wantCurrent: 0,
			wantMax:     1,
		},
		{
			name: "open current checkin does not reset early",
			checkins: []RoutineCheckin{
				streakCheckin(start, &done),
				streakCheckin(start.AddDate(0, 0, 1), nil),
			},
			wantCurrent: 1,
			wantMax:     1,
		},
		{
			name: "calendar gap resets continuity",
			checkins: []RoutineCheckin{
				streakCheckin(start, &done),
				streakCheckin(start.AddDate(0, 0, 2), &done),
			},
			wantCurrent: 1,
			wantMax:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			current, max := routineStreaks(tt.checkins)
			if current != tt.wantCurrent || max != tt.wantMax {
				t.Fatalf("streaks current=%d max=%d, want current=%d max=%d", current, max, tt.wantCurrent, tt.wantMax)
			}
		})
	}
}

func streakCheckin(date time.Time, status *domain.RoutineItemStatus) RoutineCheckin {
	return RoutineCheckin{
		CheckinDate: date,
		Items:       []RoutineCheckinItem{{Status: status}},
	}
}
