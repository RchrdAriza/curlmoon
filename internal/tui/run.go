package tui

import (
	"curlmoon/internal/collection"

	"github.com/jesseduffield/gocui"
)

// Run builds the persistence-backed app and drives it through gocui's main
// loop until the user quits.
func Run(store *collection.Store) error {
	a := NewAppWithStore(store)

	g := gocui.NewGui()
	if err := g.Init(); err != nil {
		return err
	}
	defer g.Close()

	g.Cursor = true
	// Without this, gocui puts termbox in "Alt" input mode, where a lone
	// ESC byte is held back waiting to see if it's the start of an
	// Alt+<key> combo — so pressing plain Esc (e.g. to cancel the sidebar
	// prompt) never resolves to KeyEsc at all. InputEsc mode reports it
	// as KeyEsc whenever it doesn't complete a recognized escape sequence.
	g.InputEsc = true
	g.Editor = &appEditor{app: a}
	g.SetLayout(newLayoutFunc(a))

	if err := setupKeybindings(g, a); err != nil {
		return err
	}

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		return err
	}
	return nil
}
