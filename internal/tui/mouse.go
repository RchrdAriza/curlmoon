package tui

import (
	"github.com/jesseduffield/gocui"
	termbox "github.com/nsf/termbox-go"
)

// gocui v0.3.0 only exports MouseLeft/Middle/Right, but the termbox layer it
// sits on also decodes wheel events and gocui forwards every mouse event to
// the keybindings on the view under the pointer (see Gui.onKey's EventMouse
// branch). So we bind the wheel by wrapping the termbox constants in gocui's
// Key type directly. In Termux these fire on a scroll/two-finger swipe once
// mouse reporting is on — the same gesture that scrolls lazygit.
const (
	mouseWheelUp   = gocui.Key(termbox.MouseWheelUp)
	mouseWheelDown = gocui.Key(termbox.MouseWheelDown)
)

// wheelStep is how many lines one wheel notch scrolls a text view.
const wheelStep = 2

// setupMouseBindings wires tap/click support so panels can be focused by
// touch — the same thing lazygit does with g.Mouse. gocui delivers a mouse
// event as a MouseLeft "keypress" on whichever view sits under the tap (it
// also SetCursors that view first, see Gui.onKey), so each panel just needs
// a MouseLeft binding that pulls focus to itself. Enabling g.Mouse lives in
// Run(); this only registers the handlers.
//
// Requires g.Mouse = true to have any effect.
func setupMouseBindings(g *gocui.Gui, a *App) error {
	// focusPanel mirrors keybindings.go's jumpToPanel: sync any live edits
	// out, drop out of the content sub-editor, then focus the tapped panel.
	focusPanel := func(name string) gocui.KeybindingHandler {
		return func(g *gocui.Gui, v *gocui.View) error {
			syncFromViews(g, a)
			if a.subFocus {
				a.ExitContentEditor()
			}
			a.activePanel = name
			if sv, err := g.View("status"); err == nil {
				renderStatus(sv, a)
			}
			return g.SetCurrentView(name)
		}
	}

	// sidebarTap focuses the sidebar and selects the tapped row. gocui has
	// already moved the view cursor to the tap position, so the cursor's y
	// (plus the scroll offset) is the row that was touched. Tapping the
	// already-selected row a second time activates it (open request / toggle
	// folder), so touch users never need the keyboard to drill in.
	sidebarTap := func(g *gocui.Gui, v *gocui.View) error {
		syncFromViews(g, a)
		if a.subFocus {
			a.ExitContentEditor()
		}
		alreadyHere := a.activePanel == panelSidebar
		a.activePanel = panelSidebar
		_, cy := v.Cursor()
		idx := a.sidebarOff + cy
		if idx >= 0 && idx < len(a.sidebar) {
			if alreadyHere && idx == a.sidebarSel {
				return sidebarActivate(g, a, v)
			}
			a.sidebarSel = idx
		}
		renderSidebar(v, a)
		if sv, err := g.View("status"); err == nil {
			renderStatus(sv, a)
		}
		return g.SetCurrentView(panelSidebar)
	}

	// contentTap focuses the request-content sub-editor (Headers/Body/...)
	// the same way the urlEnterContent key does.
	contentTap := func(g *gocui.Gui, v *gocui.View) error {
		syncFromViews(g, a)
		if !a.EnterContentEditor() {
			return nil
		}
		a.activePanel = panelURL
		if sv, err := g.View("status"); err == nil {
			renderStatus(sv, a)
		}
		return nil
	}

	// tabsTap focuses the URL panel and switches to whichever tab was
	// touched, so the Headers/Body/Auth/Params/Scripts row is tappable.
	tabsTap := func(g *gocui.Gui, v *gocui.View) error {
		syncFromViews(g, a)
		cx, _ := v.Cursor()
		if tab := tabAtX(cx); tab >= 0 {
			a.activeTab = tab
			loadContentTab(g, a)
			renderTabs(v, a)
		}
		if a.subFocus {
			a.ExitContentEditor()
		}
		a.activePanel = panelURL
		if sv, err := g.View("status"); err == nil {
			renderStatus(sv, a)
		}
		return g.SetCurrentView(panelURL)
	}

	binds := []struct {
		view string
		h    gocui.KeybindingHandler
	}{
		{panelSidebar, sidebarTap},
		{"method", focusPanel(panelURL)},
		{panelURL, focusPanel(panelURL)},
		{"tabs", tabsTap},
		{"content", contentTap},
		{panelResponse, focusPanel(panelResponse)},
	}
	for _, b := range binds {
		if err := g.SetKeybinding(b.view, gocui.MouseLeft, gocui.ModNone, b.h); err != nil {
			return err
		}
	}

	// scrollView shifts a view's origin by dy lines (clamped at the top),
	// which is how gocui scrolls any content taller than its frame —
	// response body, content editor, and the codegen/help overlays.
	scrollView := func(dy int) gocui.KeybindingHandler {
		return func(g *gocui.Gui, v *gocui.View) error {
			if v == nil {
				return nil
			}
			ox, oy := v.Origin()
			oy += dy
			if oy < 0 {
				oy = 0
			}
			return v.SetOrigin(ox, oy)
		}
	}
	// List views scroll by moving the selection rather than the raw origin,
	// so the wheel stays in sync with keyboard navigation.
	sidebarWheel := func(delta int) gocui.KeybindingHandler {
		return func(g *gocui.Gui, v *gocui.View) error {
			a.MoveSidebarSel(delta, sidebarVisibleRows(v))
			renderSidebar(v, a)
			return nil
		}
	}
	fbWheel := func(delta int) gocui.KeybindingHandler {
		return func(g *gocui.Gui, v *gocui.View) error {
			a.fbMoveSel(delta)
			return nil
		}
	}

	wheelBinds := []struct {
		view string
		up   gocui.KeybindingHandler
		down gocui.KeybindingHandler
	}{
		{panelResponse, scrollView(-wheelStep), scrollView(wheelStep)},
		{"content", scrollView(-wheelStep), scrollView(wheelStep)},
		{"codegen", scrollView(-wheelStep), scrollView(wheelStep)},
		{"help", scrollView(-wheelStep), scrollView(wheelStep)},
		{panelSidebar, sidebarWheel(-1), sidebarWheel(1)},
		{"filebrowser", fbWheel(-1), fbWheel(1)},
	}
	for _, b := range wheelBinds {
		if err := g.SetKeybinding(b.view, mouseWheelUp, gocui.ModNone, b.up); err != nil {
			return err
		}
		if err := g.SetKeybinding(b.view, mouseWheelDown, gocui.ModNone, b.down); err != nil {
			return err
		}
	}
	return nil
}

// sidebarActivate runs the same open/toggle logic as the sidebarEnter key
// binding, so a second tap on a selected row opens the request (or expands a
// folder). Kept in sync with the sidebarEnter closure in setupKeybindings.
func sidebarActivate(g *gocui.Gui, a *App, v *gocui.View) error {
	if !a.SelectSidebarEntry() {
		renderSidebar(v, a)
		return nil
	}
	if uv, err := g.View("url"); err == nil {
		setURLText(uv, a.urlValue)
	}
	if mv, err := g.View("method"); err == nil {
		renderMethod(mv, a)
	}
	renderSidebar(v, a)
	return g.SetCurrentView("url")
}

// tabAtX maps a 0-based column within the "tabs" view to a tab index, or -1
// if the tap landed on a gap. The layout mirrors renderTabs: each tab renders
// as " <name> " (len(name)+2 columns) joined by a single space separator.
func tabAtX(cx int) int {
	col := 0
	for i, name := range tabNames {
		w := len(name) + 2
		if cx >= col && cx < col+w {
			return i
		}
		col += w + 1 // +1 for the separator space between tabs
	}
	return -1
}
