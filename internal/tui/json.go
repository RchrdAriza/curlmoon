package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	jsonKeyStyle   = lipgloss.NewStyle().Foreground(primary)
	jsonStrStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF7F"))
	jsonNumStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500"))
	jsonBoolStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#BA55D3"))
	jsonNullStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	jsonPunctStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#E0E0E0"))
)

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
				out.WriteString(jsonKeyStyle.Render(str))
			} else {
				out.WriteString(jsonStrStyle.Render(str))
			}
			i = j

		case ch >= '0' && ch <= '9' || ch == '-' || ch == '.':
			j := i
			for j < len(text) && (text[j] >= '0' && text[j] <= '9' || text[j] == '.' || text[j] == '-' || text[j] == 'e' || text[j] == 'E' || text[j] == '+') {
				j++
			}
			out.WriteString(jsonNumStyle.Render(text[i:j]))
			i = j

		case strings.HasPrefix(strings.ToLower(text[i:]), "true"):
			out.WriteString(jsonBoolStyle.Render("true"))
			i += 4
		case strings.HasPrefix(strings.ToLower(text[i:]), "false"):
			out.WriteString(jsonBoolStyle.Render("false"))
			i += 5
		case strings.HasPrefix(strings.ToLower(text[i:]), "null"):
			out.WriteString(jsonNullStyle.Render("null"))
			i += 4

		case ch == '{' || ch == '}' || ch == '[' || ch == ']' || ch == ',' || ch == ':':
			out.WriteString(jsonPunctStyle.Render(string(ch)))
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
