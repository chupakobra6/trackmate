package ui

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/storage/postgres"
)

// Decode independently of the preview implementation, checking balanced markup
// and the visible Telegram text rather than raw HTML byte length.
func assertTelegramHTML(t *testing.T, value string) string {
	t.Helper()
	decoder := xml.NewDecoder(strings.NewReader("<root>" + value + "</root>"))
	var visible strings.Builder
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("invalid HTML: %v", err)
		}
		if text, ok := token.(xml.CharData); ok {
			visible.Write(text)
		}
	}
	text := visible.String()
	if !utf8.ValidString(value) {
		t.Fatal("invalid UTF-8")
	}
	if units := len(utf16.Encode([]rune(text))); units > 4096 {
		t.Fatalf("message has %d visible UTF-16 units", units)
	}
	return text
}

func TestDailyCardsAndProgressPreviewPreserveStatusAndSources(t *testing.T) {
	planID, reportID, thread := int64(201), int64(301), int64(10)
	plan := `<b>` + strings.Repeat(`План &amp; 👩🏽‍💻 `, 500) + `</b>`
	report := `<blockquote><i>` + strings.Repeat(`Результат &#x1f680; &lt;успех&gt; `, 500) + `</i></blockquote>`
	task := postgres.DailyTask{OwnerUserID: 42, Status: domain.DailyTaskDone, Text: plan, ReportText: &report, TaskMessageID: &planID, TaskMessageThreadID: &thread, ReportMessageID: &reportID, ReportMessageThreadID: &thread}
	payload := map[string]any{"user_id": int64(42), "display_name": "Игорь", "status": "done", "task_html": plan, "report_html": report, "task_link": "https://t.me/c/1234567890/100?thread=10", "task_source_link": "https://t.me/c/1234567890/201?thread=10", "report_link": "https://t.me/c/1234567890/301?thread=10"}
	cases := map[string]string{
		"card":                 FormatDailyTaskCard(task, "Игорь", "igor", "Подсказка", -1001234567890),
		"closed_progress":      FormatProgressEvent(postgres.ProgressEvent{EventType: domain.ProgressDailyTaskClosed, Payload: payload}),
		"auto_failed_progress": FormatProgressEvent(postgres.ProgressEvent{EventType: domain.ProgressDailyTaskAutoFail, Payload: payload}),
	}
	for name, got := range cases {
		t.Run(name, func(t *testing.T) {
			visible := assertTelegramHTML(t, got)
			for _, part := range []string{"Сокращённый просмотр. Полный текст сохранён.", "https://t.me/c/1234567890/201?thread=10", "https://t.me/c/1234567890/301?thread=10", "План:", "Результат:", "👩🏽‍💻", "🚀"} {
				if !strings.Contains(got, part) {
					t.Fatalf("missing %q", part)
				}
			}
			if !strings.Contains(visible, "Игорь") || !strings.Contains(visible, "задачу дня") {
				t.Fatal("title/status lost")
			}
			if name == "card" && !strings.Contains(got, "<blockquote>Подсказка</blockquote>") {
				t.Fatal("notice lost")
			}
		})
	}
	if task.Text != plan || *task.ReportText != report {
		t.Fatal("source text changed")
	}
	task.Kind = domain.DailyEntrySummary
	for _, got := range []string{FormatDailyTaskCard(task, "Игорь", "igor", "", -1001234567890), FormatProgressEvent(postgres.ProgressEvent{EventType: domain.ProgressDailySummaryClosed, Payload: payload})} {
		assertTelegramHTML(t, got)
		if !strings.Contains(got, `<a href="https://t.me/c/1234567890/301?thread=10">Результат</a>`) {
			t.Fatal("summary source lost")
		}
	}
}

func TestShortHTMLRemainsByteIdenticalAndMarkupDoesNotConsumeBudget(t *testing.T) {
	task := postgres.DailyTask{OwnerUserID: 42, Text: `<b>План &amp; 🐈</b>`, Status: domain.DailyTaskActive}
	want := `🎯 <b>Задача дня</b> <a href="tg://user?id=42">Игоря</a>` + "\n\n<b>План:</b>\n<blockquote><b>План &amp; 🐈</b></blockquote>"
	if got := FormatDailyTaskCard(task, "Игорь", "igor", "", -1001234567890); got != want {
		t.Fatalf("short card changed: %q", got)
	}
	fragment := `<b>` + strings.Repeat("&amp;", 4000) + `</b>`
	if got := renderPreview(previewHTML(fragment, "", "")); got != fragment {
		t.Fatal("raw HTML length incorrectly consumes visible budget")
	}
	assertTelegramHTML(t, fragment)
}

func TestPreviewFragmentKeepsEmojiClustersAndEscapedEntities(t *testing.T) {
	for _, cluster := range []string{"👩🏽‍💻", "👨‍👩‍👧‍👦", "🇷🇺", "1️⃣", "е\u0308", "&#x1f469;&#x1f3fd;‍💻"} {
		got := previewFragment(`<b>A&amp;`+cluster+`Z</b>`, 4)
		visible := assertTelegramHTML(t, got)
		if visible != "A&…" {
			t.Fatalf("split cluster %q: %q", cluster, visible)
		}
	}
	// Formatting entity boundaries can occur inside an emoji cluster.
	for _, fragment := range []string{`<b>A&amp;👩</b>🏽‍💻Z`, `<b>A&amp;👩🏽</b>‍<i>💻</i>Z`, `A&amp;<b>🇷</b>🇺Z`, `A&amp;1<b>️⃣</b>Z`} {
		if got := assertTelegramHTML(t, previewFragment(fragment, 4)); got != "A&…" {
			t.Fatalf("split cross-tag emoji: %q", got)
		}
	}
	included := previewFragment(`A<b>👩</b>🏽‍<i>💻</i>ZZZZ`, 10)
	if got := assertTelegramHTML(t, included); got != "A👩🏽‍💻Z…" {
		t.Fatalf("complete cross-tag emoji not preserved: %q", got)
	}
	// Attribute text cannot turn into visible content or be cut midway.
	got := previewFragment(`<b><a href="https://example.test/?q=>&amp;x=1">`+strings.Repeat("&amp;", 20)+`</a></b>`, 8)
	if visible := assertTelegramHTML(t, got); visible != "&&&&&&&…" {
		t.Fatalf("visible text=%q", visible)
	}
}

func TestRoutineAndGoalInputsUseBoundedPreview(t *testing.T) {
	long := strings.Repeat("Пункт & 👩🏽‍💻 ", 600)
	status := domain.RoutineItemPartial
	now := time.Date(2026, 11, 7, 0, 0, 0, 0, time.UTC)
	checkin := postgres.RoutineCheckin{CheckinDate: now, CompletedAt: &now, ReflectionText: ptr(`<b>` + strings.Repeat("Итог &amp; 🚀 ", 500) + `</b>`)}
	for i := 0; i < 9; i++ {
		checkin.Items = append(checkin.Items, postgres.RoutineCheckinItem{Text: fmt.Sprintf("Пункт %d: ", i) + long, Status: &status, ReasonText: ptr("Причина &amp; " + long)})
	}
	for _, got := range []string{FormatRoutineCheckinCard(checkin, "Игорь", "igor", "https://t.me/c/1/200?thread=10", ""), FormatRoutineCheckinFinishedCard(checkin, "Игорь", "igor", "https://t.me/c/1/200?thread=10", ""), FormatRoutineReasonPrompt(long)} {
		assertTelegramHTML(t, got)
		if !strings.Contains(got, "Сокращённый просмотр") {
			t.Fatal("missing preview notice")
		}
	}
	open := FormatRoutineCheckinCard(checkin, "Игорь", "igor", "https://t.me/c/1/200?thread=10", "")
	for i := 0; i < 9; i++ {
		if !strings.Contains(open, fmt.Sprintf("🔸 Пункт %d: ", i)) {
			t.Fatalf("routine item %d/status hidden by preview", i)
		}
	}
	source, thread := int64(200), int64(10)
	goals := FormatSeasonalGoalCard(postgres.SeasonalGoalSet{PeriodTitle: "Осень 2026", PeriodEndsOn: now, GoalsText: strings.Repeat("Цель &amp; 🚀 ", 600), SourceMessageID: &source, SourceMessageThreadID: &thread}, "Игорь", "igor", "", -1001234567890)
	assertTelegramHTML(t, goals)
	if !strings.Contains(goals, `<a href="https://t.me/c/1234567890/200?thread=10">Цели</a>`) {
		t.Fatal("goals source lost")
	}
}
func ptr(value string) *string { return &value }
