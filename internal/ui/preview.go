package ui

import (
	"html"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/igor/trackmate/internal/messages"
)

// Telegram counts message text after parsing entities. UTF-16 is a conservative
// budget for supplementary characters; markup and link destinations cost no text.
const telegramMessageUnits = 4096

type messagePart struct {
	html        string
	preview     bool
	sourceLabel string
	sourceURL   string
}

func fixedHTML(value string) messagePart { return messagePart{html: value} }
func previewHTML(value, label, source string) messagePart {
	return messagePart{html: value, preview: true, sourceLabel: label, sourceURL: source}
}

// Bound source sections before composing the message. The immutable title,
// status, notices and source links are never sliced out of the final HTML.
func renderPreview(parts ...messagePart) string {
	var full strings.Builder
	for _, p := range parts {
		full.WriteString(p.html)
	}
	if htmlUnits(full.String()) <= telegramMessageUnits {
		return full.String()
	}
	footer := "\n\n" + messages.Text("preview.shortened")
	var links []string
	for _, p := range parts {
		if p.sourceURL != "" {
			links = append(links, `<a href="`+html.EscapeString(p.sourceURL)+`">`+html.EscapeString(p.sourceLabel)+`</a>`)
		}
	}
	if len(links) > 0 {
		footer += "\n" + strings.Join(links, " · ")
	}
	remaining := telegramMessageUnits - htmlUnits(footer)
	var flexible []int
	budgets := make([]int, len(parts))
	for i, p := range parts {
		if p.preview {
			flexible = append(flexible, i)
		} else {
			remaining -= htmlUnits(p.html)
		}
	}
	// Share space between plan and report, returning unused space from short sections.
	for len(flexible) > 0 {
		share := remaining / len(flexible)
		short := false
		for j := 0; j < len(flexible); {
			i := flexible[j]
			n := htmlUnits(parts[i].html)
			if n <= share {
				budgets[i] = n
				remaining -= n
				flexible = append(flexible[:j], flexible[j+1:]...)
				short = true
			} else {
				j++
			}
		}
		if !short {
			for _, i := range flexible {
				budgets[i] = remaining / len(flexible)
			}
			break
		}
	}
	var result strings.Builder
	for i, p := range parts {
		if p.preview {
			result.WriteString(previewFragment(p.html, budgets[i]))
		} else {
			result.WriteString(p.html)
		}
	}
	result.WriteString(footer)
	return result.String()
}

// walkHTML visits complete tags and decoded text from internally generated
// Telegram HTML. It respects quoted attributes, so '>' in a URL is not a boundary.
func walkHTML(value string, visit func(string, bool) bool) {
	for len(value) > 0 {
		if value[0] != '<' {
			end := strings.IndexByte(value, '<')
			if end < 0 {
				end = len(value)
			}
			if !visit(html.UnescapeString(value[:end]), false) {
				return
			}
			value = value[end:]
			continue
		}
		quote := byte(0)
		end := 1
		for ; end < len(value); end++ {
			c := value[end]
			if quote != 0 {
				if c == quote {
					quote = 0
				}
				continue
			}
			if c == '\'' || c == '"' {
				quote = c
				continue
			}
			if c == '>' {
				break
			}
		}
		if end == len(value) {
			visit(html.UnescapeString(value), false)
			return
		}
		if !visit(value[:end+1], true) {
			return
		}
		value = value[end+1:]
	}
}

func textUnits(value string) int {
	n := 0
	for _, r := range value {
		n++
		if r > 0xffff {
			n++
		}
	}
	return n
}
func htmlUnits(value string) int {
	n := 0
	walkHTML(value, func(token string, tag bool) bool {
		if !tag {
			n += textUnits(token)
		}
		return true
	})
	return n
}

func previewFragment(value string, budget int) string {
	if htmlUnits(value) <= budget {
		return value
	}
	if budget <= 0 {
		return ""
	}
	// Determine a boundary in the complete visible text first: a Telegram
	// formatting entity may start or end inside a joined emoji sequence.
	var visible strings.Builder
	walkHTML(value, func(token string, tag bool) bool {
		if !tag {
			visible.WriteString(token)
		}
		return true
	})
	remainingUnits := budget - 1 // reserve the visible ellipsis
	remainingBytes := 0
	for text := visible.String(); len(text) > 0; {
		cluster, rest := nextCluster(text)
		n := textUnits(cluster)
		if n > remainingUnits {
			break
		}
		remainingUnits -= n
		remainingBytes += len(cluster)
		text = rest
	}
	var result strings.Builder
	var stack []string
	walkHTML(value, func(token string, tag bool) bool {
		if tag {
			name := strings.TrimSuffix(strings.TrimPrefix(token, "<"), ">")
			if strings.HasPrefix(name, "/") {
				if len(stack) > 0 {
					stack = stack[:len(stack)-1]
				}
			} else if !strings.HasSuffix(name, "/") {
				name = strings.Fields(name)[0]
				stack = append(stack, name)
			}
			result.WriteString(token)
			return true
		}
		if len(token) <= remainingBytes {
			result.WriteString(html.EscapeString(token))
			remainingBytes -= len(token)
			return true
		}
		result.WriteString(html.EscapeString(token[:remainingBytes]))
		remainingBytes = 0
		return false
	})
	for i := len(stack) - 1; i >= 0; i-- {
		result.WriteString("</" + stack[i] + ">")
	}
	result.WriteString("…")
	return result.String()
}

// Keep combining marks, variation selectors, emoji modifiers, flag pairs and
// joined emoji together. Telegram offsets and UTF-8 bytes never cut a rune.
func nextCluster(value string) (string, string) {
	first, size := utf8.DecodeRuneInString(value)
	end := size
	previous := first
	regional := first >= 0x1f1e6 && first <= 0x1f1ff
	for end < len(value) {
		r, n := utf8.DecodeRuneInString(value[end:])
		extend := unicode.Is(unicode.M, r) || r == 0x200d || previous == 0x200d || (r >= 0x1f3fb && r <= 0x1f3ff) || (r >= 0xe0020 && r <= 0xe007f)
		if regional && r >= 0x1f1e6 && r <= 0x1f1ff {
			extend = true
			regional = false
		} else if !extend {
			break
		}
		end += n
		previous = r
	}
	return value[:end], value[end:]
}
