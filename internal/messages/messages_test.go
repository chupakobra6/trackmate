package messages

import (
	"strings"
	"testing"
)

func TestTextLoadsCatalog(t *testing.T) {
	if got := Text("button.dismiss"); got != "👀 Понял" {
		t.Fatalf("button.dismiss = %q", got)
	}
	if !strings.Contains(Text("routine.plan.prompt"), "— зарядка") {
		t.Fatalf("routine prompt should show dash example: %s", Text("routine.plan.prompt"))
	}
}

func TestFormatReplacesPlaceholders(t *testing.T) {
	got := Format("daily.card.title", "emoji", "🎯", "person", "@igor")
	if got != "🎯 <b>Задача дня</b> @igor" {
		t.Fatalf("formatted title = %q", got)
	}
}

func TestParseCatalogIgnoresEditorialComments(t *testing.T) {
	values, err := parseCatalog("## greeting\n<!-- Видит пользователь при тесте. -->\nПривет\n")
	if err != nil {
		t.Fatalf("parse catalog with comments: %v", err)
	}
	if got := values["greeting"]; got != "Привет" {
		t.Fatalf("comment leaked into message: %q", got)
	}
}

func TestCatalogEntriesHaveEditorialComments(t *testing.T) {
	raw, err := catalogFS.ReadFile("messages.md")
	if err != nil {
		t.Fatalf("read catalog: %v", err)
	}
	lines := strings.Split(string(raw), "\n")
	for i, line := range lines {
		if !strings.HasPrefix(line, "## ") {
			continue
		}
		key := strings.TrimSpace(strings.TrimPrefix(line, "## "))
		if i+1 >= len(lines) || !isEditorialComment(lines[i+1]) {
			t.Fatalf("%s should have an editorial comment after the key", key)
		}
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
