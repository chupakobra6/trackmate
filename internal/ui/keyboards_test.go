package ui

import "testing"

func TestDismissKeyboardCarriesOwner(t *testing.T) {
	keyboard := DismissKeyboard(42)
	if keyboard == nil || len(keyboard.InlineKeyboard) != 1 || len(keyboard.InlineKeyboard[0]) != 1 {
		t.Fatalf("unexpected dismiss keyboard: %+v", keyboard)
	}
	if got := keyboard.InlineKeyboard[0][0].CallbackData; got != "notice:dismiss:42" {
		t.Fatalf("dismiss callback = %q", got)
	}
}
