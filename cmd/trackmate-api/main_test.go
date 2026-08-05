package main

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/igor/trackmate/internal/telegram"
)

func TestProcessUpdatesAdvancesOffsetOnlyAfterSuccess(t *testing.T) {
	wantErr := errors.New("temporary storage failure")
	var handled []int64
	offset, err := processUpdates(context.Background(), 10, []telegram.Update{
		{UpdateID: 10},
		{UpdateID: 11},
		{UpdateID: 12},
	}, func(_ context.Context, update telegram.Update) error {
		handled = append(handled, update.UpdateID)
		if update.UpdateID == 11 {
			return wantErr
		}
		return nil
	})

	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
	if offset != 11 {
		t.Fatalf("offset = %d, want 11", offset)
	}
	if !reflect.DeepEqual(handled, []int64{10, 11}) {
		t.Fatalf("handled = %v, want [10 11]", handled)
	}
}

func TestProcessUpdatesSkipsAlreadyAcknowledgedUpdates(t *testing.T) {
	var handled []int64
	offset, err := processUpdates(context.Background(), 12, []telegram.Update{
		{UpdateID: 10},
		{UpdateID: 11},
		{UpdateID: 12},
	}, func(_ context.Context, update telegram.Update) error {
		handled = append(handled, update.UpdateID)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if offset != 13 {
		t.Fatalf("offset = %d, want 13", offset)
	}
	if !reflect.DeepEqual(handled, []int64{12}) {
		t.Fatalf("handled = %v, want [12]", handled)
	}
}
