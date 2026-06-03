package telegram

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestSendMessageRetriesTransientNetworkError(t *testing.T) {
	attempts := 0
	client := &Client{
		token: "test-token",
		http: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				attempts++
				if attempts == 1 {
					return nil, errors.New("net/http: TLS handshake timeout")
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"ok": true,
						"result": {
							"message_id": 42,
							"date": 1,
							"chat": {"id": -1001, "type": "supergroup"}
						}
					}`)),
					Header: make(http.Header),
				}, nil
			}),
		},
	}

	message, err := client.SendMessage(context.Background(), SendMessageRequest{
		ChatID: -1001,
		Text:   "hello",
	})
	if err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
	if message.MessageID != 42 {
		t.Fatalf("message id = %d, want 42", message.MessageID)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}
