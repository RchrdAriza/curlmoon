package tui

import "strings"

// defaultScriptsText seeds the Scripts tab with the two sections it parses.
const defaultScriptsText = "### pre-request\n\n### test\n"

const (
	scriptPreRequestDelim = "### pre-request"
	scriptTestDelim       = "### test"
)

// parseScripts splits the Scripts tab's raw text into its pre-request and
// test sections, delimited by "### pre-request" / "### test" marker lines
// (in either order; either section may be omitted).
func parseScripts(text string) (preRequest, test string) {
	lines := strings.Split(text, "\n")

	sections := map[string][]string{}
	current := ""
	for _, line := range lines {
		trimmed := strings.ToLower(strings.TrimSpace(line))
		switch trimmed {
		case scriptPreRequestDelim:
			current = scriptPreRequestDelim
			continue
		case scriptTestDelim:
			current = scriptTestDelim
			continue
		}
		if current != "" {
			sections[current] = append(sections[current], line)
		}
	}

	preRequest = strings.TrimSpace(strings.Join(sections[scriptPreRequestDelim], "\n"))
	test = strings.TrimSpace(strings.Join(sections[scriptTestDelim], "\n"))
	return
}
