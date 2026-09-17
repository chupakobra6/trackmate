package telegram

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptrace"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestSendMessageDoesNotRepeatAmbiguousDelivery(t *testing.T) {
	attempts := 0
	client := &Client{
		token: "test-token",
		http: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				attempts++
				httptrace.ContextClientTrace(req.Context()).WroteHeaders()
				if attempts == 1 {
					return nil, errors.New("unexpected EOF after request write")
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

	_, err := client.SendMessage(context.Background(), SendMessageRequest{ChatID: -1001, Text: "hello"})
	var apiErr *Error
	if !errors.As(err, &apiErr) || !apiErr.Ambiguous {
		t.Fatalf("expected typed ambiguous error, got %v", err)
	}
	if strings.Contains(err.Error(), "test-token") {
		t.Fatal("token leaked")
	}
	if attempts != 1 {
		t.Fatalf("ambiguous send repeated %d times", attempts)
	}
}

func TestSendMessageRetriesFailureBeforeRequestWrite(t *testing.T) {
	attempts := 0
	client := &Client{token: "test-token", http: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		if attempts == 1 {
			return nil, errors.New("connection refused")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true,"result":{"message_id":42}}`)), Header: make(http.Header)}, nil
	})}}
	message, err := client.SendMessage(context.Background(), SendMessageRequest{ChatID: -1001, Text: "hello"})
	if err != nil || message.MessageID != 42 || attempts != 2 {
		t.Fatalf("safe retry message=%+v attempts=%d err=%v", message, attempts, err)
	}
}
