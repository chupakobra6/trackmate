package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/igor/trackmate/internal/observability"
)

type Client struct {
	token  string
	http   *http.Client
	logger *slog.Logger
}

type Error struct {
	Method      string
	StatusCode  int
	Description string
}

func (e *Error) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("telegram %s failed: %d %s", e.Method, e.StatusCode, e.Description)
	}
	return fmt.Sprintf("telegram %s failed: %s", e.Method, e.Description)
}

func NewClient(token string, logger *slog.Logger) *Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     false,
		MaxIdleConns:          20,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: time.Second,
	}
	return &Client{
		token: token,
		http: &http.Client{
			Timeout:   80 * time.Second,
			Transport: transport,
		},
		logger: logger,
	}
}

func (c *Client) PollUpdates(ctx context.Context, offset int64, timeoutSeconds int) ([]Update, error) {
	body := map[string]any{
		"offset":          offset,
		"timeout":         timeoutSeconds,
		"allowed_updates": []string{"message", "callback_query", "my_chat_member"},
	}
	var response getUpdatesResponse
	if err := c.callJSON(ctx, "getUpdates", body, &response); err != nil {
		return nil, err
	}
	return response.Result, nil
}

func (c *Client) AnswerCallbackQuery(ctx context.Context, request AnswerCallbackQueryRequest) error {
	var response baseResponse
	return c.callJSON(ctx, "answerCallbackQuery", request, &response)
}

func (c *Client) SendMessage(ctx context.Context, request SendMessageRequest) (Message, error) {
	if request.ParseMode == "" {
		request.ParseMode = "HTML"
	}
	var response messageResponse
	if err := c.callJSON(ctx, "sendMessage", request, &response); err != nil {
		return Message{}, err
	}
	if c.logger != nil {
		c.logger.InfoContext(ctx, "telegram_send_message_completed", observability.LogAttrs(ctx,
			"chat_id", request.ChatID,
			"message_id", response.Result.MessageID,
			"thread_id", request.MessageThreadID,
		)...)
	}
	return response.Result, nil
}

func (c *Client) EditMessageText(ctx context.Context, request EditMessageTextRequest) error {
	if request.ParseMode == "" {
		request.ParseMode = "HTML"
	}
	var response baseResponse
	if err := c.callJSON(ctx, "editMessageText", request, &response); err != nil {
		if IsNotModifiedError(err) {
			return nil
		}
		return err
	}
	return nil
}

func (c *Client) DeleteMessage(ctx context.Context, chatID int64, messageID int64) error {
	if messageID == 0 {
		return nil
	}
	var response baseResponse
	err := c.callJSON(ctx, "deleteMessage", map[string]any{"chat_id": chatID, "message_id": messageID}, &response)
	if IsMissingDeleteTarget(err) {
		return nil
	}
	return err
}

func (c *Client) PinChatMessage(ctx context.Context, chatID int64, messageID int64) error {
	var response baseResponse
	return c.callJSON(ctx, "pinChatMessage", map[string]any{
		"chat_id":              chatID,
		"message_id":           messageID,
		"disable_notification": true,
	}, &response)
}

func (c *Client) GetMe(ctx context.Context) (User, error) {
	var response userResponse
	if err := c.callJSON(ctx, "getMe", map[string]any{}, &response); err != nil {
		return User{}, err
	}
	return response.Result, nil
}

func (c *Client) GetChat(ctx context.Context, chatID int64) (Chat, error) {
	var response chatResponse
	if err := c.callJSON(ctx, "getChat", map[string]any{"chat_id": chatID}, &response); err != nil {
		return Chat{}, err
	}
	return response.Result, nil
}

func (c *Client) GetChatMember(ctx context.Context, chatID int64, userID int64) (ChatMember, error) {
	var response chatMemberResponse
	if err := c.callJSON(ctx, "getChatMember", map[string]any{"chat_id": chatID, "user_id": userID}, &response); err != nil {
		return ChatMember{}, err
	}
	return response.Result, nil
}

func (c *Client) CreateForumTopic(ctx context.Context, request CreateForumTopicRequest) (ForumTopic, error) {
	var response forumTopicResponse
	if err := c.callJSON(ctx, "createForumTopic", request, &response); err != nil {
		return ForumTopic{}, err
	}
	return response.Result, nil
}

func (c *Client) EditForumTopic(ctx context.Context, request EditForumTopicRequest) error {
	var response baseResponse
	if err := c.callJSON(ctx, "editForumTopic", request, &response); err != nil {
		if IsNotModifiedError(err) {
			return nil
		}
		return err
	}
	return nil
}

func (c *Client) callJSON(ctx context.Context, method string, request any, dest any) error {
	encoded, err := json.Marshal(request)
	if err != nil {
		return err
	}
	var lastErr error
	attempts := retryAttempts(method)
	for attempt := 1; attempt <= attempts; attempt++ {
		if attempt > 1 {
			if err := sleepContext(ctx, retryDelay(attempt)); err != nil {
				return err
			}
		}
		if err := c.callJSONOnce(ctx, method, encoded, dest); err != nil {
			lastErr = err
		} else {
			return nil
		}
		if attempt == attempts || !IsTransientRequestError(lastErr) {
			return lastErr
		}
		if c.logger != nil {
			c.logger.WarnContext(ctx, "telegram_request_retrying", observability.LogAttrs(ctx, "method", method, "attempt", attempt, "error", lastErr)...)
		}
	}
	return lastErr
}

func (c *Client) callJSONOnce(ctx context.Context, method string, encoded []byte, dest any) error {
	requestCtx, cancel := context.WithTimeout(ctx, requestTimeout(method))
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, c.apiURL(method), bytes.NewReader(encoded))
	if err != nil {
		return c.redactError(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if method != "getUpdates" {
		req.Close = true
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return c.redactError(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return c.redactError(&Error{Method: method, StatusCode: resp.StatusCode, Description: string(body)})
	}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(dest); err != nil {
		return err
	}
	if description := responseDescription(dest); description != "" {
		return &Error{Method: method, Description: description}
	}
	return nil
}

func responseDescription(dest any) string {
	switch v := dest.(type) {
	case *baseResponse:
		if !v.OK {
			return v.Description
		}
	case *getUpdatesResponse:
		if !v.OK {
			return "getUpdates not ok"
		}
	case *messageResponse:
		if !v.OK {
			return v.Description
		}
	case *userResponse:
		if !v.OK {
			return v.Description
		}
	case *chatResponse:
		if !v.OK {
			return v.Description
		}
	case *chatMemberResponse:
		if !v.OK {
			return v.Description
		}
	case *forumTopicResponse:
		if !v.OK {
			return v.Description
		}
	}
	return ""
}

func retryAttempts(method string) int {
	switch method {
	case "getUpdates", "answerCallbackQuery", "editMessageText", "deleteMessage", "pinChatMessage", "getChat", "getChatMember", "editForumTopic":
		return 3
	default:
		return 1
	}
}

func retryDelay(attempt int) time.Duration {
	switch attempt {
	case 2:
		return 250 * time.Millisecond
	case 3:
		return 750 * time.Millisecond
	default:
		return 0
	}
}

func requestTimeout(method string) time.Duration {
	switch method {
	case "getUpdates":
		return 70 * time.Second
	case "answerCallbackQuery":
		return 8 * time.Second
	case "sendMessage":
		return 60 * time.Second
	case "editMessageText":
		return 20 * time.Second
	case "deleteMessage":
		return 15 * time.Second
	case "pinChatMessage", "createForumTopic", "editForumTopic":
		return 30 * time.Second
	default:
		return 30 * time.Second
	}
}

func (c *Client) apiURL(method string) string {
	return fmt.Sprintf("https://api.telegram.org/bot%s/%s", c.token, method)
}

func (c *Client) redactError(err error) error {
	if err == nil || c.token == "" {
		return err
	}
	return errors.New(strings.ReplaceAll(err.Error(), c.token, "<redacted-token>"))
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func IsNotModifiedError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "not modified") || strings.Contains(text, "topic_not_modified")
}

func IsMissingThreadError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "message thread not found") || strings.Contains(text, "topic_id_invalid")
}

func IsMissingDeleteTarget(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "message to delete not found") ||
		strings.Contains(text, "message can't be deleted") ||
		strings.Contains(text, "message_id_invalid")
}

func IsTransientRequestError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "timeout") ||
		strings.Contains(text, "connection reset") ||
		strings.Contains(text, "connection refused") ||
		strings.Contains(text, "bad gateway") ||
		strings.Contains(text, "too many requests") ||
		strings.Contains(text, "gateway timeout") ||
		strings.Contains(text, "temporarily unavailable") ||
		strings.Contains(text, "eof")
}
