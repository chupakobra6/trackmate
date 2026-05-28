package telegram

import (
	"context"
	"time"
)

type API interface {
	PollUpdates(ctx context.Context, offset int64, timeoutSeconds int) ([]Update, error)
	AnswerCallbackQuery(ctx context.Context, request AnswerCallbackQueryRequest) error
	SendMessage(ctx context.Context, request SendMessageRequest) (Message, error)
	EditMessageText(ctx context.Context, request EditMessageTextRequest) error
	DeleteMessage(ctx context.Context, chatID int64, messageID int64) error
	PinChatMessage(ctx context.Context, chatID int64, messageID int64) error
	GetMe(ctx context.Context) (User, error)
	GetChat(ctx context.Context, chatID int64) (Chat, error)
	GetChatMember(ctx context.Context, chatID int64, userID int64) (ChatMember, error)
	CreateForumTopic(ctx context.Context, request CreateForumTopicRequest) (ForumTopic, error)
	EditForumTopic(ctx context.Context, request EditForumTopicRequest) error
}

type Update struct {
	UpdateID     int64              `json:"update_id"`
	Message      *Message           `json:"message,omitempty"`
	Callback     *CallbackQuery     `json:"callback_query,omitempty"`
	MyChatMember *ChatMemberUpdated `json:"my_chat_member,omitempty"`
}

type ChatMemberUpdated struct {
	Chat          Chat       `json:"chat"`
	From          User       `json:"from"`
	Date          int64      `json:"date"`
	OldChatMember ChatMember `json:"old_chat_member"`
	NewChatMember ChatMember `json:"new_chat_member"`
}

type CallbackQuery struct {
	ID      string   `json:"id"`
	From    User     `json:"from"`
	Message *Message `json:"message,omitempty"`
	Data    string   `json:"data,omitempty"`
}

type Message struct {
	MessageID       int64           `json:"message_id"`
	MessageThreadID int64           `json:"message_thread_id,omitempty"`
	DateUnix        int64           `json:"date"`
	From            *User           `json:"from,omitempty"`
	Chat            Chat            `json:"chat"`
	Text            string          `json:"text,omitempty"`
	Entities        []MessageEntity `json:"entities,omitempty"`
	Caption         string          `json:"caption,omitempty"`
	CaptionEntities []MessageEntity `json:"caption_entities,omitempty"`
	MediaGroupID    string          `json:"media_group_id,omitempty"`
	Voice           *Voice          `json:"voice,omitempty"`
	VideoNote       *VideoNote      `json:"video_note,omitempty"`
	Video           *Video          `json:"video,omitempty"`
	Photo           []PhotoSize     `json:"photo,omitempty"`
	Audio           *Audio          `json:"audio,omitempty"`
	Document        *Document       `json:"document,omitempty"`
	Animation       *Animation      `json:"animation,omitempty"`
	Sticker         *Sticker        `json:"sticker,omitempty"`
	Contact         *Contact        `json:"contact,omitempty"`
	Location        *Location       `json:"location,omitempty"`
	Venue           *Venue          `json:"venue,omitempty"`
	Poll            *Poll           `json:"poll,omitempty"`
	Dice            *Dice           `json:"dice,omitempty"`
	Game            *Game           `json:"game,omitempty"`
	Invoice         *Invoice        `json:"invoice,omitempty"`
	ForwardOrigin   *ForwardOrigin  `json:"forward_origin,omitempty"`
}

func (m Message) Date() time.Time {
	if m.DateUnix == 0 {
		return time.Now().UTC()
	}
	return time.Unix(m.DateUnix, 0).UTC()
}

type Chat struct {
	ID      int64  `json:"id"`
	Type    string `json:"type"`
	Title   string `json:"title,omitempty"`
	IsForum bool   `json:"is_forum,omitempty"`
}

type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

type ChatMember struct {
	Status          string `json:"status"`
	User            User   `json:"user"`
	CanManageTopics bool   `json:"can_manage_topics,omitempty"`
}

type MessageEntity struct {
	Type   string `json:"type"`
	Offset int    `json:"offset"`
	Length int    `json:"length"`
	URL    string `json:"url,omitempty"`
}

type ForwardOrigin struct {
	Type      string `json:"type"`
	Chat      *Chat  `json:"chat,omitempty"`
	MessageID int64  `json:"message_id,omitempty"`
}

type Voice struct{}
type VideoNote struct{}
type Video struct{}
type PhotoSize struct{}
type Animation struct{}
type Sticker struct {
	Emoji string `json:"emoji,omitempty"`
}
type Audio struct {
	Title     string `json:"title,omitempty"`
	Performer string `json:"performer,omitempty"`
}
type Document struct {
	FileName string `json:"file_name,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
}
type Contact struct {
	FirstName   string `json:"first_name,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`
}
type Location struct {
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
}
type Venue struct {
	Title   string `json:"title,omitempty"`
	Address string `json:"address,omitempty"`
}
type Poll struct {
	Question string `json:"question,omitempty"`
}
type Dice struct {
	Emoji string `json:"emoji,omitempty"`
	Value int    `json:"value,omitempty"`
}
type Game struct {
	Title string `json:"title,omitempty"`
}
type Invoice struct {
	Title string `json:"title,omitempty"`
}

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
}

type SendMessageRequest struct {
	ChatID                int64                 `json:"chat_id"`
	MessageThreadID       int64                 `json:"message_thread_id,omitempty"`
	Text                  string                `json:"text"`
	ParseMode             string                `json:"parse_mode,omitempty"`
	ReplyMarkup           *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
	ReplyToMessageID      int64                 `json:"reply_to_message_id,omitempty"`
	DisableNotification   bool                  `json:"disable_notification,omitempty"`
	DisableWebPagePreview *bool                 `json:"disable_web_page_preview,omitempty"`
}

type EditMessageTextRequest struct {
	ChatID      int64                 `json:"chat_id"`
	MessageID   int64                 `json:"message_id"`
	Text        string                `json:"text"`
	ParseMode   string                `json:"parse_mode,omitempty"`
	ReplyMarkup *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

type AnswerCallbackQueryRequest struct {
	CallbackQueryID string `json:"callback_query_id"`
	Text            string `json:"text,omitempty"`
	ShowAlert       bool   `json:"show_alert,omitempty"`
}

type CreateForumTopicRequest struct {
	ChatID int64  `json:"chat_id"`
	Name   string `json:"name"`
}

type EditForumTopicRequest struct {
	ChatID          int64  `json:"chat_id"`
	MessageThreadID int64  `json:"message_thread_id"`
	Name            string `json:"name"`
}

type ForumTopic struct {
	MessageThreadID int64  `json:"message_thread_id"`
	Name            string `json:"name,omitempty"`
}

type getUpdatesResponse struct {
	OK     bool     `json:"ok"`
	Result []Update `json:"result"`
}

type messageResponse struct {
	OK          bool    `json:"ok"`
	Description string  `json:"description,omitempty"`
	Result      Message `json:"result"`
}

type userResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
	Result      User   `json:"result"`
}

type chatResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
	Result      Chat   `json:"result"`
}

type chatMemberResponse struct {
	OK          bool       `json:"ok"`
	Description string     `json:"description,omitempty"`
	Result      ChatMember `json:"result"`
}

type forumTopicResponse struct {
	OK          bool       `json:"ok"`
	Description string     `json:"description,omitempty"`
	Result      ForumTopic `json:"result"`
}

type baseResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
}
