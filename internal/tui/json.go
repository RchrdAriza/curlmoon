package tui

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/jesseduffield/gocui"
)

// prettyJSON reindents JSON text for display. Requests bodies are sent
// exactly as typed (this is display-only), so invalid JSON is returned
// unchanged rather than erroring.
func prettyJSON(text string) string {
	if !looksLikeJSON(text) {
		return text
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(text), "", "  "); err != nil {
		return text
	}
	return buf.String()
}

func highlightJSON(text string) string {
	if !looksLikeJSON(text) {
		return text
	}

	var out strings.Builder
	i := 0
	for i < len(text) {
		ch := text[i]
		switch {
		case ch == '"':
			j := i + 1
			escaped := false
			for j < len(text) {
				if text[j] == '\\' {
					escaped = !escaped
				} else if text[j] == '"' && !escaped {
					j++
					break
				} else {
					escaped = false
				}
				j++
			}
			str := text[i:j]
			if j < len(text) && text[j] == ':' {
				out.WriteString(ansiWrap(str, currentTheme.Primary, false))
			} else {
				out.WriteString(ansiWrap(str, currentTheme.Success, false))
			}
			i = j

		case ch >= '0' && ch <= '9' || ch == '-' || ch == '.':
			j := i
			for j < len(text) && (text[j] >= '0' && text[j] <= '9' || text[j] == '.' || text[j] == '-' || text[j] == 'e' || text[j] == 'E' || text[j] == '+') {
				j++
			}
			out.WriteString(ansiWrap(text[i:j], currentTheme.Secondary, false))
			i = j

		case strings.HasPrefix(strings.ToLower(text[i:]), "true"):
			out.WriteString(ansiWrap("true", currentTheme.Magenta, false))
			i += 4
		case strings.HasPrefix(strings.ToLower(text[i:]), "false"):
			out.WriteString(ansiWrap("false", currentTheme.Magenta, false))
			i += 5
		case strings.HasPrefix(strings.ToLower(text[i:]), "null"):
			out.WriteString(ansiWrap("null", currentTheme.Muted, false))
			i += 4

		case ch == '{' || ch == '}' || ch == '[' || ch == ']' || ch == ',' || ch == ':':
			out.WriteString(ansiWrap(string(ch), gocui.ColorWhite, false))
			i++

		default:
			out.WriteByte(ch)
			i++
		}
	}
	return out.String()
}

func looksLikeJSON(text string) bool {
	t := strings.TrimSpace(text)
	return strings.HasPrefix(t, "{") || strings.HasPrefix(t, "[")
}
