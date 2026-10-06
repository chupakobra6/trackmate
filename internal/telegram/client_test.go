package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptrace"
	"strings"
	"testing"
	"time"
)

func TestSendMessageOnceNeverRetriesAndDefaultsToHTML(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusBadGateway, http.StatusGatewayTimeout} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			attempts := 0
			client := &Client{token: "test-token", http: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				attempts++
				var request SendMessageRequest
				if err := json.NewDecoder(req.Body).Decode(&request); err != nil || request.ParseMode != "HTML" {
					t.Fatalf("expected HTML request: %+v err=%v", request, err)
				}
				body := `{"ok":true,"result":{"message_id":42}}`
				if status != http.StatusOK {
					body = http.StatusText(status)
				}
				return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
			})}}
			message, err := client.SendMessageOnce(context.Background(), SendMessageRequest{ChatID: -1001, Text: "hello"})
			if attempts != 1 || (status == http.StatusOK && (err != nil || message.MessageID != 42)) || (status != http.StatusOK && err == nil) {
				t.Fatalf("one-shot message=%+v attempts=%d err=%v", message, attempts, err)
			}
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestRateLimitHonorsServerDelayAndRetainsCooldownAfterExhaustion(t *testing.T) {
	var requests []time.Time
	client := &Client{token: "test-token", http: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests = append(requests, time.Now())
		return &http.Response{StatusCode: 429, Body: io.NopCloser(strings.NewReader(`{"ok":false,"parameters":{"retry_after":1}}`)), Header: make(http.Header)}, nil
	})}}
	err := client.EditMessageText(context.Background(), EditMessageTextRequest{ChatID: -1001, MessageID: 42, Text: "hello"})
	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.RetryAfter != time.Second {
		t.Fatalf("expected server retry delay, got %v", err)
	}
	if len(requests) != 3 {
		t.Fatalf("expected bounded three attempts, got %d", len(requests))
	}
	for i := 1; i < len(requests); i++ {
		if delay := requests[i].Sub(requests[i-1]); delay < time.Second {
			t.Fatalf("request %d repeated early after %v", i+1, delay)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err = client.EditMessageText(ctx, EditMessageTextRequest{ChatID: -1001, MessageID: 42, Text: "hello again"})
	if !errors.Is(err, context.DeadlineExceeded) || len(requests) != 3 {
		t.Fatalf("new call bypassed exhausted rate limit: attempts=%d err=%v", len(requests), err)
	}
}

func TestRateLimitWaitCanBeCanceled(t *testing.T) {
	attempts := 0
	client := &Client{token: "test-token", http: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		return &http.Response{StatusCode: 429, Body: io.NopCloser(strings.NewReader(`{"ok":false,"parameters":{"retry_after":30}}`)), Header: make(http.Header)}, nil
	})}}
	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := client.EditMessageText(ctx, EditMessageTextRequest{ChatID: -1001, MessageID: 42, Text: "hello"})
	if !errors.Is(err, context.DeadlineExceeded) || attempts != 1 || time.Since(start) > 2*time.Second {
		t.Fatalf("rate limit cancellation attempts=%d elapsed=%v err=%v", attempts, time.Since(start), err)
	}
}

func TestRateLimitIgnoresMalformedOrNonpositiveDelay(t *testing.T) {
	for _, body := range []string{`invalid json`, `{"parameters":{"retry_after":0}}`, `{"parameters":{"retry_after":-1}}`} {
		t.Run(body, func(t *testing.T) {
			client := &Client{token: "test-token", http: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 429, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
			})}}
			var response baseResponse
			err := client.callJSONOnce(context.Background(), "editMessageText", []byte(`{}`), &response)
			var apiErr *Error
			if !errors.As(err, &apiErr) || apiErr.RetryAfter != 0 {
				t.Fatalf("unexpected retry delay: %v", err)
			}
		})
	}
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
