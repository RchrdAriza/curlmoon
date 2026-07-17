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
