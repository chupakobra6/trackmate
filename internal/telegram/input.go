package telegram

import (
	"fmt"
	"html"
	"sort"
	"strings"
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

func MessageInputText(message Message) string {
	if message.Text != "" {
		return message.Text
	}
	if message.Caption != "" {
		return message.Caption
	}
	switch {
	case message.Voice != nil:
		return "Голосовое сообщение"
	case message.VideoNote != nil:
		return "Видео-кружок"
	case message.Video != nil:
		return "Видео"
	case len(message.Photo) > 0:
		return "Фото"
	case message.Audio != nil:
		if message.Audio.Title != "" && message.Audio.Performer != "" {
			return fmt.Sprintf("Аудио: %s - %s", message.Audio.Performer, message.Audio.Title)
		}
		if message.Audio.Title != "" {
			return "Аудио: " + message.Audio.Title
		}
		return "Аудио"
	case message.Document != nil:
		if message.Document.FileName != "" {
			return "Документ: " + message.Document.FileName
		}
		return "Документ"
	case message.Animation != nil:
		return "Анимация"
	case message.Sticker != nil:
		if message.Sticker.Emoji != "" {
			return "Стикер " + message.Sticker.Emoji
		}
		return "Стикер"
	case message.Contact != nil:
		if message.Contact.FirstName != "" && message.Contact.PhoneNumber != "" {
			return fmt.Sprintf("Контакт: %s (%s)", message.Contact.FirstName, message.Contact.PhoneNumber)
		}
		if message.Contact.FirstName != "" {
			return "Контакт: " + message.Contact.FirstName
		}
		return "Контакт"
	case message.Location != nil:
		return fmt.Sprintf("Локация: %v, %v", message.Location.Latitude, message.Location.Longitude)
	case message.Venue != nil:
		if message.Venue.Title != "" && message.Venue.Address != "" {
			return fmt.Sprintf("Место: %s, %s", message.Venue.Title, message.Venue.Address)
		}
		if message.Venue.Title != "" {
			return "Место: " + message.Venue.Title
		}
		return "Место"
	case message.Poll != nil:
		if message.Poll.Question != "" {
			return "Опрос: " + message.Poll.Question
		}
		return "Опрос"
	case message.Dice != nil:
		if message.Dice.Emoji != "" && message.Dice.Value != 0 {
			return fmt.Sprintf("Кубик %s: %d", message.Dice.Emoji, message.Dice.Value)
		}
		return "Кубик"
	case message.Game != nil:
		if message.Game.Title != "" {
			return "Игра: " + message.Game.Title
		}
		return "Игра"
	case message.Invoice != nil:
		if message.Invoice.Title != "" {
			return "Счет: " + message.Invoice.Title
		}
		return "Счет"
	default:
		return "Сообщение"
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
