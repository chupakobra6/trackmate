package domain

import "testing"

func TestParseCallbackTaskStatus(t *testing.T) {
	cb, err := ParseCallback("task:status:42:done")
	if err != nil {
		t.Fatal(err)
	}
	if cb.Kind != CallbackTaskStatus || cb.TaskID != 42 || cb.TaskStatus != DailyTaskDone {
		t.Fatalf("unexpected callback: %+v", cb)
	}
}

func TestParseCallbackRejectsUnknownStatus(t *testing.T) {
	if _, err := ParseCallback("task:status:42:wat"); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseCallbackRejectsRemovedMaterialsCallbacks(t *testing.T) {
	for _, raw := range []string{"material:read:10", "material:note:10", "material:applied:10", "material:unknown:10", "material:bad"} {
		cb, err := ParseCallback(raw)
		if err == nil {
			t.Fatalf("%s: expected error", raw)
		}
		if cb.Kind != CallbackUnknown {
			t.Fatalf("%s: expected unknown callback, got %s", raw, cb.Kind)
		}
	}
}

func TestParseRoutineItemCallback(t *testing.T) {
	got, err := ParseCallback("routine:item:42:2:partial")
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != CallbackRoutineItem || got.RoutineCheckinID != 42 || got.RoutineItemIndex != 2 || got.RoutineItemStatus != RoutineItemPartial {
		t.Fatalf("unexpected callback: %+v", got)
	}
}

func TestParseGoalFinalCallback(t *testing.T) {
	got, err := ParseCallback("goals:final:77:done")
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != CallbackGoalFinalStatus || got.GoalSetID != 77 || got.GoalFinalStatus != GoalFinalDone {
		t.Fatalf("unexpected callback: %+v", got)
	}
}
