package telegram

import (
	"fmt"
	"html"
	"sort"
	"strings"

	"github.com/igor/trackmate/internal/messages"
)

func DisplayName(user User) string {
	name := strings.TrimSpace(strings.TrimSpace(user.FirstName) + " " + strings.TrimSpace(user.LastName))
	if name != "" {
		return name
	}
	if user.Username != "" {
		return user.Username
	}
	return fmt.Sprintf("%d", user.ID)
}

func MessageInputKind(message Message) string {
	if message.Text != "" || message.Caption != "" {
		return "text"
	}
	return "non_text"
}

type MessageInput struct {
	Kind     string
	TextHTML string
	Source   MessageSource
}

type MessageSource struct {
	ThreadID  int64
	MessageID int64
	UserID    int64
}

func NewMessageInput(message Message) MessageInput {
	var userID int64
	if message.From != nil {
		userID = message.From.ID
	}
	return MessageInput{
		Kind:     MessageInputKind(message),
		TextHTML: MessageInputHTML(message),
		Source: MessageSource{
			ThreadID:  message.MessageThreadID,
			MessageID: message.MessageID,
			UserID:    userID,
		},
	}
}

func MessageInputText(message Message) string {
	if message.Text != "" {
		return message.Text
	}
	if message.Caption != "" {
		return message.Caption
	}
	switch {
	case message.Voice != nil:
		return messages.Text("input.voice")
	case message.VideoNote != nil:
		return messages.Text("input.video_note")
	case message.Video != nil:
		return messages.Text("input.video")
	case len(message.Photo) > 0:
		return messages.Text("input.photo")
	case message.Audio != nil:
		if message.Audio.Title != "" && message.Audio.Performer != "" {
			return fmt.Sprintf("%s: %s - %s", messages.Text("input.audio"), message.Audio.Performer, message.Audio.Title)
		}
		if message.Audio.Title != "" {
			return messages.Text("input.audio") + ": " + message.Audio.Title
		}
		return messages.Text("input.audio")
	case message.Document != nil:
		if message.Document.FileName != "" {
			return messages.Text("input.document") + ": " + message.Document.FileName
		}
		return messages.Text("input.document")
	case message.Animation != nil:
		return messages.Text("input.animation")
	case message.Sticker != nil:
		if message.Sticker.Emoji != "" {
			return messages.Text("input.sticker") + " " + message.Sticker.Emoji
		}
		return messages.Text("input.sticker")
	case message.Contact != nil:
		if message.Contact.FirstName != "" && message.Contact.PhoneNumber != "" {
			return fmt.Sprintf("%s: %s (%s)", messages.Text("input.contact"), message.Contact.FirstName, message.Contact.PhoneNumber)
		}
		if message.Contact.FirstName != "" {
			return messages.Text("input.contact") + ": " + message.Contact.FirstName
		}
		return messages.Text("input.contact")
	case message.Location != nil:
		return fmt.Sprintf("%s: %v, %v", messages.Text("input.location"), message.Location.Latitude, message.Location.Longitude)
	case message.Venue != nil:
		if message.Venue.Title != "" && message.Venue.Address != "" {
			return fmt.Sprintf("%s: %s, %s", messages.Text("input.venue"), message.Venue.Title, message.Venue.Address)
		}
		if message.Venue.Title != "" {
			return messages.Text("input.venue") + ": " + message.Venue.Title
		}
		return messages.Text("input.venue")
	case message.Poll != nil:
		if message.Poll.Question != "" {
			return messages.Text("input.poll") + ": " + message.Poll.Question
		}
		return messages.Text("input.poll")
	case message.Dice != nil:
		if message.Dice.Emoji != "" && message.Dice.Value != 0 {
			return fmt.Sprintf("%s %s: %d", messages.Text("input.dice"), message.Dice.Emoji, message.Dice.Value)
		}
		return messages.Text("input.dice")
	case message.Game != nil:
		if message.Game.Title != "" {
			return messages.Text("input.game") + ": " + message.Game.Title
		}
		return messages.Text("input.game")
	case message.Invoice != nil:
		if message.Invoice.Title != "" {
			return messages.Text("input.invoice") + ": " + message.Invoice.Title
		}
		return messages.Text("input.invoice")
	default:
		return messages.Text("input.message")
	}
}

func MessageInputHTML(message Message) string {
	if message.Text != "" {
		return renderEntitiesHTML(message.Text, message.Entities)
	}
	if message.Caption != "" {
		return renderEntitiesHTML(message.Caption, message.CaptionEntities)
	}
	return html.EscapeString(MessageInputText(message))
}

func renderEntitiesHTML(text string, entities []MessageEntity) string {
	if text == "" {
		return ""
	}
	if len(entities) == 0 {
		return html.EscapeString(text)
	}
	runes := []rune(text)
	unitToRune := utf16UnitToRuneIndex(runes)
	type tag struct {
		pos    int
		text   string
		closer bool
	}
	tags := make([]tag, 0, len(entities)*2)
	for _, entity := range entities {
		start, ok := unitToRune[entity.Offset]
		if !ok {
			continue
		}
		end, ok := unitToRune[entity.Offset+entity.Length]
		if !ok {
			continue
		}
		open, close := entityTags(entity)
		if open == "" && close == "" {
			continue
		}
		tags = append(tags, tag{pos: start, text: open}, tag{pos: end, text: close, closer: true})
	}
	sort.SliceStable(tags, func(i, j int) bool {
		if tags[i].pos == tags[j].pos {
			return tags[i].closer && !tags[j].closer
		}
		return tags[i].pos < tags[j].pos
	})
	var builder strings.Builder
	tagIndex := 0
	for i, r := range runes {
		for tagIndex < len(tags) && tags[tagIndex].pos == i {
			builder.WriteString(tags[tagIndex].text)
			tagIndex++
		}
		builder.WriteString(html.EscapeString(string(r)))
	}
	for tagIndex < len(tags) {
		builder.WriteString(tags[tagIndex].text)
		tagIndex++
	}
	return builder.String()
}

func entityTags(entity MessageEntity) (string, string) {
	switch entity.Type {
	case "bold":
		return "<b>", "</b>"
	case "italic":
		return "<i>", "</i>"
	case "underline":
		return "<u>", "</u>"
	case "strikethrough":
		return "<s>", "</s>"
	case "spoiler":
		return "<tg-spoiler>", "</tg-spoiler>"
	case "code":
		return "<code>", "</code>"
	case "pre":
		return "<pre>", "</pre>"
	case "blockquote":
		return "<blockquote>", "</blockquote>"
	case "text_link":
		return fmt.Sprintf(`<a href="%s">`, html.EscapeString(entity.URL)), "</a>"
	default:
		return "", ""
	}
}

func utf16UnitToRuneIndex(runes []rune) map[int]int {
	result := map[int]int{0: 0}
	units := 0
	for index, r := range runes {
		if r <= 0xFFFF {
			units++
		} else {
			units += 2
		}
		result[units] = index + 1
	}
	return result
}
