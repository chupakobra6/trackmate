package telegram

import "testing"

func TestMessageInputHTMLPreservesSimpleEntities(t *testing.T) {
	message := Message{
		Text: "read docs",
		Entities: []MessageEntity{
			{Type: "text_link", Offset: 5, Length: 4, URL: "https://platform.openai.com/docs"},
		},
	}
	got := MessageInputHTML(message)
	want := `read <a href="https://platform.openai.com/docs">docs</a>`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestMessageInputTextSupportsNonTextFallbacks(t *testing.T) {
	got := MessageInputText(Message{Document: &Document{FileName: "report.pdf"}})
	if got != "Документ: report.pdf" {
		t.Fatalf("unexpected document fallback: %q", got)
	}
}

func TestTelegramErrorClassifiers(t *testing.T) {
	if !IsMissingThreadError(&Error{Description: "Bad Request: message thread not found"}) {
		t.Fatal("missing thread was not classified")
	}
	if !IsNotModifiedError(&Error{Description: "Bad Request: TOPIC_NOT_MODIFIED"}) {
		t.Fatal("not modified was not classified")
	}
	if !IsMissingDeleteTarget(&Error{Description: "Bad Request: message to delete not found"}) {
		t.Fatal("missing delete was not classified")
	}
}
