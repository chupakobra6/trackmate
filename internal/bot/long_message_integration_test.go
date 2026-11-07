package bot_test

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
	"unicode/utf16"

	appprogress "github.com/igor/trackmate/internal/app/progress"
	"github.com/igor/trackmate/internal/bot"
	"github.com/igor/trackmate/internal/domain"
	"github.com/igor/trackmate/internal/logging"
	"github.com/igor/trackmate/internal/telegram"
	"github.com/igor/trackmate/internal/testsupport"
)

func TestLongTaskReportAndSourceEditKeepFullStorageAndSingleCards(t *testing.T) {
	store, _ := testsupport.OpenMigratedStore(t)
	ctx := context.Background()
	q := store.Queries()
	fake := newFakeTelegram()
	service := bot.NewService(store, fake, logging.New("ERROR"), "UTC", 99)
	workspace, err := q.GetOrCreateWorkspace(ctx, -1001234567890, "Group", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	for kind, thread := range map[domain.TopicKey]int64{domain.TopicToday: 10, domain.TopicProgress: 11} {
		if _, err := q.UpsertTopicBinding(ctx, workspace.ID, kind, thread, string(kind)); err != nil {
			t.Fatal(err)
		}
	}
	participant, err := q.RegisterParticipant(ctx, workspace.ID, 42, "igor", "Игорь")
	if err != nil {
		t.Fatal(err)
	}
	plan := `<b>` + strings.Repeat("План &amp; 👩🏽‍💻 ", 200) + `</b>`
	task, created, err := q.CreateDailyTask(ctx, workspace.ID, participant.ID, 42, time.Now().UTC(), plan, 201, 20)
	if err != nil || !created {
		t.Fatalf("created=%v err=%v", created, err)
	}
	if err := q.SetDailyTaskCardMessageID(ctx, task.ID, 100); err != nil {
		t.Fatal(err)
	}
	if _, err := q.UpsertPendingInput(ctx, workspace.ID, 42, 10, domain.PendingDailyTaskReport, map[string]any{"task_id": task.ID, "status": string(domain.DailyTaskDone), "prompt_message_id": 101}); err != nil {
		t.Fatal(err)
	}
	report := telegram.Message{MessageID: 301, MessageThreadID: 10, DateUnix: time.Now().Unix(), From: &telegram.User{ID: 42, Username: "igor", FirstName: "Игорь"}, Chat: telegram.Chat{ID: workspace.ChatID, Type: "supergroup", IsForum: true}, Text: strings.Repeat("Итог & 🚀 ", 300)}
	report.Entities = []telegram.MessageEntity{{Type: "italic", Offset: 0, Length: len(utf16.Encode([]rune(report.Text)))}}
	if _, err := service.HandleUpdate(ctx, telegram.Update{UpdateID: 1, Message: &report}); err != nil {
		t.Fatal(err)
	}
	if err := appprogress.PublishPending(ctx, store, fake); err != nil {
		t.Fatal(err)
	}
	if len(fake.sent) != 1 {
		t.Fatalf("expected one Progress message, got %d", len(fake.sent))
	}
	assertLongMessagePreview(t, fake.sent[0].Text)
	if edit, ok := fake.findEdit(100); !ok {
		t.Fatal("Today card missing")
	} else {
		assertLongMessagePreview(t, edit.Text)
	}
	// Updating either original preserves the same Today and Progress message IDs.
	report.Text = strings.Repeat("Правка & 🚀 ", 300)
	report.Entities[0].Length = len(utf16.Encode([]rune(report.Text)))
	if _, err := service.HandleUpdate(ctx, telegram.Update{UpdateID: 2, EditedMessage: &report}); err != nil {
		t.Fatal(err)
	}
	stored, found, err := q.GetTask(ctx, task.ID)
	if err != nil || !found || stored.Text != plan || stored.ReportText == nil || *stored.ReportText != telegram.MessageInputHTML(report) {
		t.Fatalf("full source text not retained: found=%v err=%v", found, err)
	}
	if stored.TodayCardMessageID == nil || *stored.TodayCardMessageID != 100 {
		t.Fatal("Today message ID changed")
	}
	var taskHTML, reportHTML, sourceLink, reportLink string
	var progressID int64
	if err := store.Pool().QueryRow(ctx, `SELECT payload->>'task_html',payload->>'report_html',payload->>'task_source_link',payload->>'report_link',published_message_id FROM progress_events WHERE daily_task_id=$1`, task.ID).Scan(&taskHTML, &reportHTML, &sourceLink, &reportLink, &progressID); err != nil {
		t.Fatal(err)
	}
	if taskHTML != plan || reportHTML != *stored.ReportText || sourceLink != "https://t.me/c/1234567890/201?thread=20" || reportLink != "https://t.me/c/1234567890/301?thread=10" {
		t.Fatal("progress source payload lost full text or original topic links")
	}
	if progressID != fake.nextMessageID {
		t.Fatal("Progress message ID changed")
	}
	for _, id := range []int64{100, progressID} {
		var latest string
		for _, edit := range fake.edits {
			if edit.MessageID == id {
				latest = edit.Text
			}
		}
		assertLongMessagePreview(t, latest)
		if !strings.Contains(latest, "Правка &amp;") {
			t.Fatalf("message %d not refreshed", id)
		}
	}
	if err := appprogress.PublishPending(ctx, store, fake); err != nil {
		t.Fatal(err)
	}
	if len(fake.sent) != 1 {
		t.Fatal("edit/republication created duplicate Progress messages")
	}
	for _, deleted := range fake.deleted {
		if deleted == 201 || deleted == 301 || deleted == 100 || deleted == progressID {
			t.Fatalf("deleted persistent source/card %d", deleted)
		}
	}
}

func assertLongMessagePreview(t *testing.T, value string) {
	t.Helper()
	decoder := xml.NewDecoder(strings.NewReader("<root>" + value + "</root>"))
	var text strings.Builder
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if data, ok := token.(xml.CharData); ok {
			text.Write(data)
		}
	}
	if units := len(utf16.Encode([]rune(text.String()))); units > 4096 {
		t.Fatalf("message has %d UTF-16 units", units)
	}
	for _, part := range []string{"Сокращённый просмотр. Полный текст сохранён.", fmt.Sprintf("https://t.me/c/%d/201?thread=20", 1234567890), "https://t.me/c/1234567890/301?thread=10"} {
		if !strings.Contains(value, part) {
			t.Fatalf("missing preview part %q", part)
		}
	}
}
