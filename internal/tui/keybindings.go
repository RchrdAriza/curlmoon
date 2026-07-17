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
			return nil
		}
		if uv, err := g.View("url"); err == nil {
			setViewText(uv, a.urlValue)
		}
		if mv, err := g.View("method"); err == nil {
			renderMethod(mv, a)
		}
		renderSidebar(v, a)
		return g.SetCurrentView("url")
	}
	sidebarNewCollection := func(g *gocui.Gui, v *gocui.View) error {
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
		a.StartPrompt("newRequest", sidebarEntry{collIdx: a.sidebar[a.sidebarSel].collIdx}, "")
		return nil
	}
	sidebarRename := func(g *gocui.Gui, v *gocui.View) error {
		if a.store == nil || len(a.sidebar) == 0 {
			return nil
		}
		sel := a.sidebar[a.sidebarSel]
		a.StartPrompt("rename", sel, sel.name)
		return nil
	}
	sidebarDelete := func(g *gocui.Gui, v *gocui.View) error {
		if a.store == nil || len(a.sidebar) == 0 {
			return nil
		}
		sel := a.sidebar[a.sidebarSel]
		a.StartPrompt("confirmDelete", sel, "")
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

	urlBindings := []struct {
		key interface{}
		h   gocui.KeybindingHandler
	}{
		{gocui.KeyArrowUp, urlUp},
		{gocui.KeyArrowDown, urlDown},
		{gocui.KeyArrowLeft, switchTab(-1)},
		{gocui.KeyArrowRight, switchTab(1)},
		{gocui.KeyEnter, urlEnter},
		{gocui.KeyCtrlR, sendRequest},
	}
	for _, b := range urlBindings {
		if err := g.SetKeybinding(panelURL, b.key, gocui.ModNone, b.h); err != nil {
			return err
		}
	}

	// --- content (headers/body/params/auth editor) ---
	contentEsc := func(g *gocui.Gui, v *gocui.View) error {
		syncFromViews(g, a)
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
