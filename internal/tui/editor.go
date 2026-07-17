package tui

import "github.com/jesseduffield/gocui"

// appEditor is the single global gocui.Editor. Enter is suppressed on
// single-line views whose submit action is already handled by a
// keybinding, so the default editor doesn't also insert a newline.
type appEditor struct{ app *App }

func (e *appEditor) Edit(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
	if key == gocui.KeyEnter {
		switch v.Name() {
		case "url", "prompt":
			return
		}
	}
	gocui.DefaultEditor.Edit(v, key, ch, mod)
}
