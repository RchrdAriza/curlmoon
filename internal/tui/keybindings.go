package tui

import (
	"github.com/jesseduffield/gocui"
)

// setupKeybindings wires every keyboard shortcut curlmoon responds to onto
// the App instance. Handlers stay thin: they sync any live-edited text out
// of the gocui views, delegate to a pure *App method, then refresh the
// views that need to reflect the new state.
func setupKeybindings(g *gocui.Gui, a *App) error {
	quit := func(g *gocui.Gui, v *gocui.View) error {
		syncFromViews(g, a)
		a.saveSession()
		return gocui.ErrQuit
	}
	if err := g.SetKeybinding("", 'q', gocui.ModNone, quit); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}

	cycleFocus := func(g *gocui.Gui, v *gocui.View) error {
		order := []string{panelSidebar, panelURL, panelResponse}
		cur := panelSidebar
		if v != nil {
			cur = v.Name()
		}
		next := order[0]
		for i, name := range order {
			if name == cur {
				next = order[(i+1)%len(order)]
				break
			}
		}
		a.activePanel = next
		return g.SetCurrentView(next)
	}
	for _, name := range []string{panelSidebar, panelURL, panelResponse} {
		if err := g.SetKeybinding(name, gocui.KeyTab, gocui.ModNone, cycleFocus); err != nil {
			return err
		}
	}

	// jumpToPanel lets a shortcut focus a panel directly instead of
	// cycling through Tab repeatedly, mirroring lazygit's numbered-panel
	// jumps. These used to be Alt+<n>, but Alt-combo detection and a lone
	// Esc resolving to KeyEsc are mutually exclusive in termbox (see
	// g.InputEsc in run.go) — since Esc needs to work to cancel the
	// sidebar prompt, panel jumps moved to Ctrl+<letter> instead: a raw
	// single-byte control code every terminal sends identically, so it
	// doesn't depend on either input mode (same reasoning as the url
	// panel's Ctrl+J/K/P/N). Plain digits/letters are reserved for typing
	// (URL/headers/body are text editors), so this needs a modifier.
	jumpToPanel := func(name string) gocui.KeybindingHandler {
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
	jumpBindings := []struct {
		key gocui.Key
		to  string
	}{
		{gocui.KeyCtrlS, panelSidebar},
		{gocui.KeyCtrlU, panelURL},
		{gocui.KeyCtrlE, panelResponse},
	}
	for _, b := range jumpBindings {
		if err := g.SetKeybinding("", b.key, gocui.ModNone, jumpToPanel(b.to)); err != nil {
			return err
		}
	}
	jumpToContent := func(g *gocui.Gui, v *gocui.View) error {
		syncFromViews(g, a)
		if !a.EnterContentEditor() {
			return nil
		}
		a.activePanel = panelURL
		if sv, err := g.View("status"); err == nil {
			renderStatus(sv, a)
		}
		return g.SetCurrentView("content")
	}
	if err := g.SetKeybinding("", gocui.KeyCtrlB, gocui.ModNone, jumpToContent); err != nil {
		return err
	}

	sendRequest := func(g *gocui.Gui, v *gocui.View) error {
		if a.sending {
			return nil
		}
		syncFromViews(g, a)
		a.StartSending()
		if sv, err := g.View("status"); err == nil {
			renderStatus(sv, a)
		}
		go func() {
			resp, err := a.doRequest()
			g.Execute(func(g *gocui.Gui) error {
				a.HandleResponse(resp, err)
				if rv, err := g.View("response"); err == nil {
					renderResponse(rv, a)
					rv.SetOrigin(0, 0)
				}
				if sv, err := g.View("status"); err == nil {
					renderStatus(sv, a)
				}
				return nil
			})
		}()
		return nil
	}

	// --- sidebar ---
	sidebarUp := func(g *gocui.Gui, v *gocui.View) error {
		a.MoveSidebarSel(-1, sidebarVisibleRows(v))
		renderSidebar(v, a)
		return nil
	}
	sidebarDown := func(g *gocui.Gui, v *gocui.View) error {
		a.MoveSidebarSel(1, sidebarVisibleRows(v))
		renderSidebar(v, a)
		return nil
	}
	sidebarEnter := func(g *gocui.Gui, v *gocui.View) error {
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
	sidebarNewCollection := func(g *gocui.Gui, v *gocui.View) error {
		if len(a.sidebar) > 0 {
			switch a.sidebar[a.sidebarSel].section {
			case "env":
				if a.envStore == nil {
					return nil
				}
				a.StartPrompt("newEnvironment", sidebarEntry{}, "")
				return nil
			case "history":
				return nil
			}
		}
		if a.store == nil {
			return nil
		}
		a.StartPrompt("newCollection", sidebarEntry{}, "")
		return nil
	}
	sidebarNewRequest := func(g *gocui.Gui, v *gocui.View) error {
		if a.store == nil || len(a.sidebar) == 0 {
			return nil
		}
		if a.sidebar[a.sidebarSel].section != "" {
			return nil
		}
		a.StartPrompt("newRequest", sidebarEntry{collIdx: a.sidebar[a.sidebarSel].collIdx}, "")
		return nil
	}
	sidebarRename := func(g *gocui.Gui, v *gocui.View) error {
		if len(a.sidebar) == 0 {
			return nil
		}
		sel := a.sidebar[a.sidebarSel]
		if sel.section == "env" {
			if sel.isFolder || a.envStore == nil {
				return nil
			}
			a.StartPrompt("renameEnv", sel, a.environments[sel.envIdx].Name)
			return nil
		}
		if sel.section != "" || a.store == nil {
			return nil
		}
		a.StartPrompt("rename", sel, sel.name)
		return nil
	}
	sidebarDelete := func(g *gocui.Gui, v *gocui.View) error {
		if len(a.sidebar) == 0 {
			return nil
		}
		sel := a.sidebar[a.sidebarSel]
		if sel.section == "env" {
			if sel.isFolder || a.envStore == nil {
				return nil
			}
			sel.name = a.environments[sel.envIdx].Name
			a.StartPrompt("confirmDeleteEnv", sel, "")
			return nil
		}
		if sel.section != "" || a.store == nil {
			return nil
		}
		a.StartPrompt("confirmDelete", sel, "")
		return nil
	}
	sidebarEditVars := func(g *gocui.Gui, v *gocui.View) error {
		if len(a.sidebar) == 0 {
			return nil
		}
		sel := a.sidebar[a.sidebarSel]
		if sel.section != "env" || sel.isFolder {
			return nil
		}
		// Don't call g.SetCurrentView here: gocui redelivers the same
		// keystroke to the newly-focused view's editor right after this
		// handler returns, which would type the triggering key into the
		// content buffer. Instead, just flag the pending switch and let
		// layout() perform it on the next redraw (same trick used for the
		// prompt overlay).
		if !a.StartEnvEdit(sel.envIdx) {
			return nil
		}
		if sv, err := g.View("status"); err == nil {
			renderStatus(sv, a)
		}
		return nil
	}

	sidebarBindings := []struct {
		key interface{}
		h   gocui.KeybindingHandler
	}{
		{gocui.KeyArrowUp, sidebarUp},
		{'k', sidebarUp},
		{gocui.KeyArrowDown, sidebarDown},
		{'j', sidebarDown},
		{gocui.KeyEnter, sidebarEnter},
		{'n', sidebarNewCollection},
		{'a', sidebarNewRequest},
		{'r', sidebarRename},
		{'d', sidebarDelete},
		{'v', sidebarEditVars},
	}
	for _, b := range sidebarBindings {
		if err := g.SetKeybinding(panelSidebar, b.key, gocui.ModNone, b.h); err != nil {
			return err
		}
	}

	// --- url ---
	urlUp := func(g *gocui.Gui, v *gocui.View) error {
		a.CycleMethod(-1)
		if mv, err := g.View("method"); err == nil {
			renderMethod(mv, a)
		}
		return nil
	}
	urlDown := func(g *gocui.Gui, v *gocui.View) error {
		a.CycleMethod(1)
		if mv, err := g.View("method"); err == nil {
			renderMethod(mv, a)
		}
		return nil
	}
	switchTab := func(delta int) gocui.KeybindingHandler {
		return func(g *gocui.Gui, v *gocui.View) error {
			syncFromViews(g, a)
			if delta < 0 {
				a.PrevTab()
			} else {
				a.NextTab()
			}
			loadContentTab(g, a)
			if tv, err := g.View("tabs"); err == nil {
				renderTabs(tv, a)
			}
			return nil
		}
	}
	urlEnter := func(g *gocui.Gui, v *gocui.View) error {
		if !a.EnterContentEditor() {
			return nil
		}
		if sv, err := g.View("status"); err == nil {
			renderStatus(sv, a)
		}
		return g.SetCurrentView("content")
	}
	urlHome := func(g *gocui.Gui, v *gocui.View) error {
		_ = v.SetOrigin(0, 0)
		_ = v.SetCursor(0, 0)
		return nil
	}
	urlEnd := func(g *gocui.Gui, v *gocui.View) error {
		setURLText(v, trimTrailingNewline(v.Buffer()))
		return nil
	}

	urlBindings := []struct {
		key interface{}
		mod gocui.Modifier
		h   gocui.KeybindingHandler
	}{
		// Left/Right stay reserved for moving the cursor inside the URL
		// text (it's an editable field) — that was the actual bug: they
		// used to double as tab-switch. Up/Down keep cycling the method
		// as before (a single-line field has no vertical text to move
		// through, so there's nothing to reclaim there). Tab switching
		// moves to Ctrl+P/N, and Ctrl+K/J duplicate method-cycle, so both
		// actions also have a non-arrow shortcut: raw single-byte control
		// codes that every terminal sends identically (same mechanism as
		// Ctrl+R below) — unlike Alt+Arrow, which relies on a
		// modifier-encoded escape sequence (e.g. "ESC[1;3C") that this
		// app's terminal library doesn't parse, so it leaked as literal
		// text into the URL instead of firing the shortcut.
		// Ctrl+J/K echo vim's j/k (down/up); Ctrl+N/P echo emacs'
		// next/previous. H/I/M are avoided: they alias Backspace/Tab/Enter.
		{gocui.KeyArrowUp, gocui.ModNone, urlUp},
		{gocui.KeyArrowDown, gocui.ModNone, urlDown},
		{gocui.KeyCtrlK, gocui.ModNone, urlUp},
		{gocui.KeyCtrlJ, gocui.ModNone, urlDown},
		{gocui.KeyCtrlP, gocui.ModNone, switchTab(-1)},
		{gocui.KeyCtrlN, gocui.ModNone, switchTab(1)},
		{gocui.KeyHome, gocui.ModNone, urlHome},
		{gocui.KeyEnd, gocui.ModNone, urlEnd},
		{gocui.KeyEnter, gocui.ModNone, urlEnter},
		{gocui.KeyCtrlR, gocui.ModNone, sendRequest},
	}
	for _, b := range urlBindings {
		if err := g.SetKeybinding(panelURL, b.key, b.mod, b.h); err != nil {
			return err
		}
	}

	// --- content (headers/body/params/auth editor) ---
	contentEsc := func(g *gocui.Gui, v *gocui.View) error {
		syncFromViews(g, a)
		if a.envEditIdx >= 0 {
			a.SaveEnvEdit()
			if sv, err := g.View("status"); err == nil {
				renderStatus(sv, a)
			}
			if sbv, err := g.View("sidebar"); err == nil {
				renderSidebar(sbv, a)
			}
			return g.SetCurrentView(panelSidebar)
		}
		a.ExitContentEditor()
		if sv, err := g.View("status"); err == nil {
			renderStatus(sv, a)
		}
		return g.SetCurrentView("url")
	}
	if err := g.SetKeybinding("content", gocui.KeyEsc, gocui.ModNone, contentEsc); err != nil {
		return err
	}
	if err := g.SetKeybinding("content", gocui.KeyCtrlR, gocui.ModNone, sendRequest); err != nil {
		return err
	}

	// --- response (scroll) ---
	scrollResponse := func(dy int) gocui.KeybindingHandler {
		return func(g *gocui.Gui, v *gocui.View) error {
			ox, oy := v.Origin()
			oy += dy
			if oy < 0 {
				oy = 0
			}
			return v.SetOrigin(ox, oy)
		}
	}
	responseBindings := []struct {
		key interface{}
		h   gocui.KeybindingHandler
	}{
		{gocui.KeyArrowUp, scrollResponse(-1)},
		{gocui.KeyArrowDown, scrollResponse(1)},
		{gocui.KeyPgup, scrollResponse(-10)},
		{gocui.KeyPgdn, scrollResponse(10)},
	}
	for _, b := range responseBindings {
		if err := g.SetKeybinding(panelResponse, b.key, gocui.ModNone, b.h); err != nil {
			return err
		}
	}

	// --- help overlay ---
	// Bound to Ctrl+/ rather than literal Ctrl+? — terminals collapse
	// Ctrl+<shifted key> onto the same single-byte control code as the
	// unshifted key (see the jumpToPanel comment above), so Ctrl+/ and
	// Ctrl+? already send an identical byte. Toggling on both Ctrl+/ and
	// Esc (when open) means the same chord opens and closes it.
	toggleHelp := func(g *gocui.Gui, v *gocui.View) error {
		if a.promptMode != "" {
			return nil
		}
		a.showHelp = !a.showHelp
		return nil
	}
	if err := g.SetKeybinding("", gocui.KeyCtrlSlash, gocui.ModNone, toggleHelp); err != nil {
		return err
	}
	if err := g.SetKeybinding("help", gocui.KeyEsc, gocui.ModNone, toggleHelp); err != nil {
		return err
	}
	if err := g.SetKeybinding("help", gocui.KeyCtrlSlash, gocui.ModNone, toggleHelp); err != nil {
		return err
	}

	// --- prompt overlay ---
	promptConfirm := func(g *gocui.Gui, v *gocui.View) error {
		if a.promptMode != "confirmDelete" {
			a.promptText = trimTrailingNewline(v.Buffer())
		}
		a.ConfirmPrompt()
		if sbv, err := g.View("sidebar"); err == nil {
			renderSidebar(sbv, a)
		}
		return nil
	}
	promptCancel := func(g *gocui.Gui, v *gocui.View) error {
		a.CancelPrompt()
		return nil
	}
	promptBindings := []struct {
		key interface{}
		h   gocui.KeybindingHandler
	}{
		{gocui.KeyEnter, promptConfirm},
		{gocui.KeyEsc, promptCancel},
		{'y', func(g *gocui.Gui, v *gocui.View) error {
			if a.promptMode == "confirmDelete" {
				return promptConfirm(g, v)
			}
			return nil
		}},
		{'n', func(g *gocui.Gui, v *gocui.View) error {
			if a.promptMode == "confirmDelete" {
				return promptCancel(g, v)
			}
			return nil
		}},
	}
	for _, b := range promptBindings {
		if err := g.SetKeybinding("prompt", b.key, gocui.ModNone, b.h); err != nil {
			return err
		}
	}

	return nil
}

// sidebarVisibleRows returns how many sidebar rows currently fit on screen,
// used to keep the scroll offset in sync with the selection.
func sidebarVisibleRows(v *gocui.View) int {
	_, h := v.Size()
	if h < 1 {
		return 1
	}
	return h
}
