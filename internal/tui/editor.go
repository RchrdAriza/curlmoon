package tui

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/gocui"
)

// appEditor is the single global gocui.Editor. gocui delivers every
// keystroke to both the matching keybinding *and* the focused view's
// editor unconditionally (see onKey in gocui's gui.go) — so views whose
// keybindings already fully own a key need it suppressed here, or the
// default editor's reaction to the same keystroke corrupts the buffer
// alongside it.
type appEditor struct{ app *App }

// bracketClosers maps each opening bracket handled by auto-close to its
// matching closer.
var bracketClosers = map[rune]rune{'{': '}', '[': ']', '(': ')'}

func (e *appEditor) Edit(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
	jsonBody := v.Name() == "content" && e.app.envEditIdx < 0 &&
		e.app.activeTab == tabBody && e.app.bodyType == 1

	if key == gocui.KeyEnter {
		switch v.Name() {
		case "url", "prompt":
			return
		}
		if jsonBody {
			jsonNewLine(v)
			recolorJSON(v)
			return
		}
	}

	if jsonBody && ch != 0 && mod == 0 {
		if closer, ok := bracketClosers[ch]; ok {
			v.EditWrite(ch)
			v.EditWrite(closer)
			v.MoveCursor(-1, 0, true)
			recolorJSON(v)
			return
		}
		if isBracketCloser(ch) {
			if _, after := cursorContext(v); after == ch {
				v.MoveCursor(1, 0, true)
				recolorJSON(v)
				return
			}
		}
	}

	if jsonBody && (key == gocui.KeyBackspace || key == gocui.KeyBackspace2) {
		if before, after := cursorContext(v); before != 0 && bracketClosers[before] == after {
			v.EditDelete(false)
			v.EditDelete(true)
			recolorJSON(v)
			return
		}
	}

	// The URL bar is single-line; Up/Down are fully repurposed to cycle
	// the HTTP method (see urlUp/urlDown in keybindings.go). Left to the
	// default editor too, they call View.MoveCursor with dy=±1 on a view
	// with no second line, which pads gocui's internal line buffer with a
	// null rune — Buffer() then renders that null as a literal trailing
	// space, silently corrupting the URL that gets sent.
	if v.Name() == "url" && (key == gocui.KeyArrowUp || key == gocui.KeyArrowDown) {
		return
	}
	gocui.DefaultEditor.Edit(v, key, ch, mod)

	if jsonBody && editModifiesText(key, ch) {
		recolorJSON(v)
	}
}

func isBracketCloser(ch rune) bool {
	switch ch {
	case '}', ']', ')':
		return true
	}
	return false
}

// cursorContext returns the runes immediately before and after the cursor
// on its current line (0 if there is none), used to decide whether typing
// a closing bracket should just skip over an existing one and whether
// Backspace is deleting an empty open/close pair in one step.
func cursorContext(v *gocui.View) (before, after rune) {
	cx, cy := v.Cursor()
	ox, _ := v.Origin()
	x := ox + cx
	line, err := v.Line(cy)
	if err != nil {
		return 0, 0
	}
	runes := []rune(line)
	if x > len(runes) {
		x = len(runes)
	}
	if x > 0 {
		before = runes[x-1]
	}
	if x < len(runes) {
		after = runes[x]
	}
	return before, after
}

// editModifiesText reports whether a keystroke DefaultEditor.Edit just
// applied could have changed the buffer's text, as opposed to only moving
// the cursor — recoloring is only worth doing in the former case.
func editModifiesText(key gocui.Key, ch rune) bool {
	if ch != 0 {
		return true
	}
	switch key {
	case gocui.KeySpace, gocui.KeyBackspace, gocui.KeyBackspace2, gocui.KeyDelete:
		return true
	}
	return false
}

// jsonNewLine inserts a newline into a JSON body view, indenting the new
// line to match the current one and adding one more indent level after an
// opening brace/bracket. If the cursor sits directly between an opener and
// its matching closer, the closer is pushed onto its own dedented line so
// the cursor lands on a blank indented line between them — the "smart"
// Enter behavior most JSON-aware editors provide.
func jsonNewLine(v *gocui.View) {
	cx, cy := v.Cursor()
	ox, _ := v.Origin()
	x := ox + cx
	line, err := v.Line(cy)
	if err != nil {
		v.EditNewLine()
		return
	}
	runes := []rune(line)
	if x > len(runes) {
		x = len(runes)
	}
	before := string(runes[:x])
	after := string(runes[x:])

	indent := leadingWhitespace(line)
	trimmedBefore := strings.TrimRight(before, " \t")
	increase := strings.HasSuffix(trimmedBefore, "{") || strings.HasSuffix(trimmedBefore, "[")
	trimmedAfter := strings.TrimLeft(after, " \t")
	closes := increase && (strings.HasPrefix(trimmedAfter, "}") || strings.HasPrefix(trimmedAfter, "]"))

	newIndent := indent
	if increase {
		newIndent += "  "
	}

	v.EditNewLine()
	for _, r := range newIndent {
		v.EditWrite(r)
	}

	if closes {
		cx2, cy2 := v.Cursor()
		ox2, oy2 := v.Origin()
		v.EditNewLine()
		for _, r := range indent {
			v.EditWrite(r)
		}
		_ = v.SetOrigin(ox2, oy2)
		_ = v.SetCursor(cx2, cy2)
	}
}

func leadingWhitespace(line string) string {
	i := 0
	for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
		i++
	}
	return line[:i]
}

// recolorJSON re-applies JSON syntax highlighting to the whole buffer after
// an edit, preserving cursor and scroll position. This only ever changes
// color, never the underlying text: ansiWrap adds SGR escapes that gocui's
// Write parses and consumes without emitting cells for them (see
// parseInput in gocui's view.go), so every visible rune keeps the same
// line/column it had before the rewrite and the saved cursor stays valid.
func recolorJSON(v *gocui.View) {
	cx, cy := v.Cursor()
	ox, oy := v.Origin()
	text := strings.TrimSuffix(v.Buffer(), "\n")
	v.Clear()
	fmt.Fprint(v, highlightJSON(text))
	_ = v.SetOrigin(ox, oy)
	_ = v.SetCursor(cx, cy)
}
