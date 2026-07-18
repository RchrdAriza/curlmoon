package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jesseduffield/gocui"
)

// fbEntry is one row in the file browser: a directory to descend into or a
// (JSON) file to import. The synthetic ".." parent entry is a dir too.
type fbEntry struct {
	name  string
	isDir bool
}

// OpenFileBrowser opens the native-feeling file explorer overlay so the user
// can pick an import source / export directory by navigating the filesystem
// instead of typing a path by hand. mode is "import" or "export"; target is
// the collection being written out (export only).
func (a *App) OpenFileBrowser(mode string, target sidebarEntry) {
	a.fbMode = mode
	a.fbTarget = target
	a.fbErr = ""
	// Start where the user is most likely to keep their files: the store's
	// base dir for export (next to existing collections), the home dir for
	// import (so any downloaded file is reachable).
	start, _ := os.UserHomeDir()
	if mode == "export" && a.store != nil {
		start = a.store.BaseDir
	}
	if start == "" {
		start, _ = os.Getwd()
	}
	a.fbNavigate(start)
}

// CloseFileBrowser dismisses the overlay without picking anything.
func (a *App) CloseFileBrowser() {
	a.fbMode = ""
	a.fbEntries = nil
	a.fbSel = 0
	a.fbOffset = 0
	a.fbErr = ""
}

// fbNavigate reads dir and makes it the current listing. On failure (e.g. a
// permission-denied directory) it keeps the previous listing and surfaces the
// error rather than leaving the browser empty.
func (a *App) fbNavigate(dir string) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = dir
	}
	items, err := os.ReadDir(abs)
	if err != nil {
		a.fbErr = fmt.Sprintf("Cannot open %s: %v", abs, err)
		return
	}
	var dirs, files []fbEntry
	for _, it := range items {
		name := it.Name()
		if strings.HasPrefix(name, ".") {
			continue // skip dotfiles/dirs to reduce noise
		}
		if it.IsDir() {
			dirs = append(dirs, fbEntry{name: name, isDir: true})
			continue
		}
		// Import only cares about JSON; export hides files entirely since
		// you're choosing a destination directory, not a file.
		if a.fbMode == "import" && strings.EqualFold(filepath.Ext(name), ".json") {
			files = append(files, fbEntry{name: name})
		}
	}
	sort.Slice(dirs, func(i, j int) bool { return strings.ToLower(dirs[i].name) < strings.ToLower(dirs[j].name) })
	sort.Slice(files, func(i, j int) bool { return strings.ToLower(files[i].name) < strings.ToLower(files[j].name) })

	entries := make([]fbEntry, 0, len(dirs)+len(files)+1)
	if parent := filepath.Dir(abs); parent != abs {
		entries = append(entries, fbEntry{name: "..", isDir: true})
	}
	entries = append(entries, dirs...)
	entries = append(entries, files...)

	a.fbDir = abs
	a.fbEntries = entries
	a.fbSel = 0
	a.fbOffset = 0
	a.fbErr = ""
}

// fbMoveSel moves the highlight by delta, clamped to the listing bounds.
func (a *App) fbMoveSel(delta int) {
	if len(a.fbEntries) == 0 {
		return
	}
	a.fbSel += delta
	if a.fbSel < 0 {
		a.fbSel = 0
	}
	if a.fbSel >= len(a.fbEntries) {
		a.fbSel = len(a.fbEntries) - 1
	}
}

// fbParent walks up one directory level.
func (a *App) fbParent() {
	if parent := filepath.Dir(a.fbDir); parent != a.fbDir {
		a.fbNavigate(parent)
	}
}

// fbEnter is the primary action on the highlighted entry: descend into a
// directory, or (import mode) import the highlighted JSON file. It returns a
// status message to show and whether the overlay should close.
func (a *App) fbEnter() (msg string, done bool) {
	if len(a.fbEntries) == 0 {
		return "", false
	}
	e := a.fbEntries[a.fbSel]
	if e.isDir {
		if e.name == ".." {
			a.fbParent()
		} else {
			a.fbNavigate(filepath.Join(a.fbDir, e.name))
		}
		return "", false
	}
	// A file — only reachable in import mode.
	path := filepath.Join(a.fbDir, e.name)
	c, err := a.store.Import(path)
	if err != nil {
		a.fbErr = fmt.Sprintf("Error: %v", err)
		return "", false
	}
	replaced := false
	for i, existing := range a.collections {
		if existing.Info.Name == c.Info.Name {
			a.collections[i] = c
			replaced = true
			break
		}
	}
	if !replaced {
		a.collections = append(a.collections, c)
	}
	return fmt.Sprintf("Imported %q from %s", c.Info.Name, path), true
}

// fbSelectHere is export's confirm action: it takes the current directory as
// the destination and hands off to the exportPath prompt (prefilled with a
// sensible filename) so the user can still tweak the file name before writing.
func (a *App) fbSelectHere() {
	if a.fbMode != "export" || a.fbTarget.collIdx >= len(a.collections) {
		return
	}
	name := a.collections[a.fbTarget.collIdx].Info.Name
	prefill := filepath.Join(a.fbDir, name+".json")
	target := a.fbTarget
	a.CloseFileBrowser()
	a.StartPrompt("exportPath", target, prefill)
}

// renderFileBrowser paints the file explorer overlay each frame.
func renderFileBrowser(v *gocui.View, a *App) {
	if a.fbMode == "export" {
		v.Title = "Export — choose a folder (Enter open · Ctrl+S save here · Esc cancel)"
	} else {
		v.Title = "Import — choose a .json file (Enter open/select · Esc cancel)"
	}

	_, viewH := v.Size()
	rows := viewH - 1 // leave a line for the path header
	if rows < 1 {
		rows = 1
	}
	// Keep the selection inside the visible window.
	if a.fbSel < a.fbOffset {
		a.fbOffset = a.fbSel
	}
	if a.fbSel >= a.fbOffset+rows {
		a.fbOffset = a.fbSel - rows + 1
	}

	var b strings.Builder
	b.WriteString(ansiWrap(a.fbDir, currentTheme.Muted, true))
	b.WriteString("\n")

	if a.fbErr != "" {
		b.WriteString(ansiWrap(a.fbErr, currentTheme.Error, true))
	} else if len(a.fbEntries) == 0 {
		b.WriteString(ansiWrap("  (empty)", currentTheme.Muted, false))
	} else {
		end := a.fbOffset + rows
		if end > len(a.fbEntries) {
			end = len(a.fbEntries)
		}
		for i := a.fbOffset; i < end; i++ {
			e := a.fbEntries[i]
			label := e.name
			if e.isDir && e.name != ".." {
				label += "/"
			}
			cursor := "  "
			if i == a.fbSel {
				cursor = ansiWrap("> ", currentTheme.Primary, true)
			}
			color := currentTheme.Muted
			bold := false
			if e.isDir {
				color = currentTheme.Primary
				bold = true
			}
			b.WriteString(cursor)
			b.WriteString(ansiWrap(label, color, bold))
			b.WriteString("\n")
		}
	}
	setViewText(v, b.String())
	v.SetOrigin(0, 0)
}
