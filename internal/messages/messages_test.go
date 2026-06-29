package messages

import (
	"strings"
	"testing"
)

func TestTextLoadsCatalog(t *testing.T) {
	if got := Text("button.dismiss"); got != "👀 Понял" {
		t.Fatalf("button.dismiss = %q", got)
	}
	if !strings.Contains(Text("routine.plan.prompt"), "- зарядка") {
		t.Fatalf("routine prompt should show dash example: %s", Text("routine.plan.prompt"))
	}
}

func TestFormatReplacesPlaceholders(t *testing.T) {
	got := Format("daily.card.title", "emoji", "🎯", "person", "@igor")
	if got != "🎯 <b>Задача дня</b> @igor" {
		t.Fatalf("formatted title = %q", got)
	}
}

func TestMultilineCatalogTextsHaveHeaderGap(t *testing.T) {
	for key, text := range All() {
		lines := strings.Split(text, "\n")
		if len(lines) < 2 {
			continue
		}
		if strings.TrimSpace(lines[0]) == "" || strings.TrimSpace(lines[1]) == "" {
			continue
		}
		t.Fatalf("%s should keep a blank line after the first line: %q", key, text)
	}
}
