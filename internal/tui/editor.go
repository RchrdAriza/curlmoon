package tui

import "github.com/jesseduffield/gocui"

// appEditor is the single global gocui.Editor. gocui delivers every
// keystroke to both the matching keybinding *and* the focused view's
// editor unconditionally (see onKey in gocui's gui.go) — so views whose
// keybindings already fully own a key need it suppressed here, or the
// default editor's reaction to the same keystroke corrupts the buffer
// alongside it.
type appEditor struct{ app *App }

func (e *appEditor) Edit(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
	if key == gocui.KeyEnter {
		switch v.Name() {
		case "url", "prompt":
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
}
