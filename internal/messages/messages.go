package messages

import (
	"embed"
	"fmt"
	"strings"
	"sync"
)

//go:embed messages.md
var catalogFS embed.FS

var (
	once    sync.Once
	catalog map[string]string
	loadErr error
)

func Text(key string) string {
	values, err := load()
	if err != nil {
		panic(err)
	}
	value, ok := values[key]
	if !ok {
		panic(fmt.Sprintf("message key %q not found", key))
	}
	return value
}

func Format(key string, pairs ...string) string {
	value := Text(key)
	if len(pairs) == 0 {
		return value
	}
	if len(pairs)%2 != 0 {
		panic("messages.Format requires key/value pairs")
	}
	replacements := make([]string, 0, len(pairs))
	for i := 0; i < len(pairs); i += 2 {
		replacements = append(replacements, "{{"+pairs[i]+"}}", pairs[i+1])
	}
	return strings.NewReplacer(replacements...).Replace(value)
}

func All() map[string]string {
	values, err := load()
	if err != nil {
		panic(err)
	}
	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func load() (map[string]string, error) {
	once.Do(func() {
		raw, err := catalogFS.ReadFile("messages.md")
		if err != nil {
			loadErr = err
			return
		}
		catalog, loadErr = parseCatalog(string(raw))
	})
	return catalog, loadErr
}

func parseCatalog(raw string) (map[string]string, error) {
	result := map[string]string{}
	var currentKey string
	var builder strings.Builder
	flush := func() error {
		if currentKey == "" {
			return nil
		}
		value := strings.Trim(builder.String(), "\n")
		if _, exists := result[currentKey]; exists {
			return fmt.Errorf("duplicate message key %q", currentKey)
		}
		result[currentKey] = value
		builder.Reset()
		return nil
	}
	for _, line := range strings.Split(raw, "\n") {
		if strings.HasPrefix(line, "## ") {
			if err := flush(); err != nil {
				return nil, err
			}
			currentKey = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			if currentKey == "" {
				return nil, fmt.Errorf("empty message key")
			}
			continue
		}
		if isEditorialComment(line) {
			continue
		}
		if currentKey == "" {
			continue
		}
		builder.WriteString(line)
		builder.WriteByte('\n')
	}
	if err := flush(); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("message catalog is empty")
	}
	return result, nil
}

func isEditorialComment(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "<!--") && strings.HasSuffix(trimmed, "-->")
}
