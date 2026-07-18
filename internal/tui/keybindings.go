package tui

import (
	"fmt"

	"github.com/jesseduffield/gocui"
)

// bind resolves action's key through a.keymap (falling back to its default
// if unset or unparsable) and registers it on view. This is the only place
// keybindings.go touches raw key literals for user-configurable actions —
// everything else goes through here so ~/.curlmoon/keybindings.json can
// override it.
func bind(g *gocui.Gui, a *App, view, action string, h gocui.KeybindingHandler) error {
	key, mod := a.keymap.Key(action)
	return g.SetKeybinding(view, key, mod, h)
}

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
	// quit is scoped to the non-editable views only (never "url", "content",
	// or "prompt" while it accepts text) — gocui delivers a "" (global)
	// keybinding's handler on every keystroke *in addition to* the view's
	// own text editor, regardless of focus, so a global 'q' would quit the
	// app the instant anyone typed the letter q into a URL, header, or
	// script (e.g. a GraphQL "query { ... }").
	for _, view := range []string{panelSidebar, panelResponse, "help", "codegen"} {
		if err := bind(g, a, view, "quit", quit); err != nil {
			return err
		}
	}
	// Ctrl+C always force-quits, regardless of keymap config — a fixed
	// safety net so a broken keybindings.json can never lock the app open.
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
		if err := bind(g, a, name, "cycleFocus", cycleFocus); err != nil {
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
		action string
		to     string
	}{
		{"jumpSidebar", panelSidebar},
		{"jumpURL", panelURL},
		{"jumpResponse", panelResponse},
	}
	for _, b := range jumpBindings {
		if err := bind(g, a, "", b.action, jumpToPanel(b.to)); err != nil {
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
		return nil
	}
	if err := bind(g, a, "", "jumpContent", jumpToContent); err != nil {
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
	sidebarExport := func(g *gocui.Gui, v *gocui.View) error {
		if a.store == nil || len(a.sidebar) == 0 {
			return nil
		}
		sel := a.sidebar[a.sidebarSel]
		if sel.section != "" || len(sel.itemPath) != 0 {
			return nil
		}
		name := a.collections[sel.collIdx].Info.Name
		a.StartPrompt("exportPath", sel, a.store.BaseDir+"/"+name+".json")
		return nil
	}
	sidebarImport := func(g *gocui.Gui, v *gocui.View) error {
		if a.store == nil {
			return nil
		}
		a.StartPrompt("importPath", sidebarEntry{}, "")
		return nil
	}

	sidebarBindings := []struct {
		action string
		h      gocui.KeybindingHandler
	}{
		{"sidebarUp", sidebarUp},
		{"sidebarDown", sidebarDown},
		{"sidebarEnter", sidebarEnter},
		{"sidebarNewCollection", sidebarNewCollection},
		{"sidebarNewRequest", sidebarNewRequest},
		{"sidebarRename", sidebarRename},
		{"sidebarDelete", sidebarDelete},
		{"sidebarEditVars", sidebarEditVars},
		{"sidebarExport", sidebarExport},
		{"sidebarImport", sidebarImport},
	}
	for _, b := range sidebarBindings {
		if err := bind(g, a, panelSidebar, b.action, b.h); err != nil {
			return err
		}
	}
	// Vim-style j/k aliases for sidebar navigation always work alongside
	// whatever the configured up/down keys are — a fixed convenience, not
	// itself user-configurable.
	if err := g.SetKeybinding(panelSidebar, 'k', gocui.ModNone, sidebarUp); err != nil {
		return err
	}
	if err := g.SetKeybinding(panelSidebar, 'j', gocui.ModNone, sidebarDown); err != nil {
		return err
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
		return nil
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
	cycleBodyType := func(delta int) gocui.KeybindingHandler {
		return func(g *gocui.Gui, v *gocui.View) error {
			if a.activeTab != tabBody {
				return nil
			}
			a.CycleBodyType(delta)
			if sv, err := g.View("status"); err == nil {
				a.statusMsg = fmt.Sprintf("Body type: %s", bodyTypes[a.bodyType])
				renderStatus(sv, a)
			}
			return nil
		}
	}

	// Left/Right stay reserved for moving the cursor inside the URL text
	// (it's an editable field) — not configurable. Ctrl+K/J and Ctrl+P/N
	// stay as fixed aliases (vim/emacs muscle memory) alongside whatever
	// the configured urlMethodUp/Down and urlSwitchTabPrev/Next keys are,
	// same rationale as the sidebar's j/k aliases above.
	urlBindings := []struct {
		action string
		h      gocui.KeybindingHandler
	}{
		{"urlMethodUp", urlUp},
		{"urlMethodDown", urlDown},
		{"urlSwitchTabPrev", switchTab(-1)},
		{"urlSwitchTabNext", switchTab(1)},
		{"urlHome", urlHome},
		{"urlEnd", urlEnd},
		{"urlEnterContent", urlEnter},
		{"sendRequest", sendRequest},
		{"cycleBodyType", cycleBodyType(1)},
	}
	for _, b := range urlBindings {
		if err := bind(g, a, panelURL, b.action, b.h); err != nil {
			return err
		}
	}
	// Ctrl+K/J are fixed vim-style aliases for method cycling, alongside
	// whatever urlMethodUp/Down resolve to (arrows by default) — same
	// rationale as the sidebar's j/k aliases above. Tab-switching has no
	// such alias: urlSwitchTabPrev/Next (Ctrl+P/N by default) are its only
	// binding.
	fixedURLBindings := []struct {
		key gocui.Key
		h   gocui.KeybindingHandler
	}{
		{gocui.KeyCtrlK, urlUp},
		{gocui.KeyCtrlJ, urlDown},
	}
	for _, b := range fixedURLBindings {
		if err := g.SetKeybinding(panelURL, b.key, gocui.ModNone, b.h); err != nil {
			return err
		}
	}

	// --- content (headers/body/params/auth/scripts editor) ---
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
	if err := bind(g, a, "content", "contentEsc", contentEsc); err != nil {
		return err
	}
	if err := bind(g, a, "content", "sendRequest", sendRequest); err != nil {
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
		action string
		h      gocui.KeybindingHandler
	}{
		{"responseScrollUp", scrollResponse(-1)},
		{"responseScrollDown", scrollResponse(1)},
		{"responsePageUp", scrollResponse(-10)},
		{"responsePageDown", scrollResponse(10)},
	}
	for _, b := range responseBindings {
		if err := bind(g, a, panelResponse, b.action, b.h); err != nil {
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
	if err := bind(g, a, "", "toggleHelp", toggleHelp); err != nil {
		return err
	}
	if err := g.SetKeybinding("help", gocui.KeyEsc, gocui.ModNone, toggleHelp); err != nil {
		return err
	}
	if err := g.SetKeybinding("help", gocui.KeyCtrlSlash, gocui.ModNone, toggleHelp); err != nil {
		return err
	}

	// --- code generation overlay ---
	toggleCodegen := func(g *gocui.Gui, v *gocui.View) error {
		if a.promptMode != "" || a.showHelp {
			return nil
		}
		syncFromViews(g, a)
		a.ToggleCodegen()
		return nil
	}
	if err := bind(g, a, "", "toggleCodegen", toggleCodegen); err != nil {
		return err
	}
	codegenNextLang := func(g *gocui.Gui, v *gocui.View) error {
		a.NextCodegenLang()
		return nil
	}
	codegenPrevLang := func(g *gocui.Gui, v *gocui.View) error {
		a.PrevCodegenLang()
		return nil
	}
	if err := g.SetKeybinding("codegen", gocui.KeyTab, gocui.ModNone, codegenNextLang); err != nil {
		return err
	}
	if err := g.SetKeybinding("codegen", gocui.KeyBackspace2, gocui.ModNone, codegenPrevLang); err != nil {
		return err
	}

	// --- theme toggle ---
	toggleTheme := func(g *gocui.Gui, v *gocui.View) error {
		a.ToggleTheme()
		return nil
	}
	if err := bind(g, a, "", "toggleTheme", toggleTheme); err != nil {
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
