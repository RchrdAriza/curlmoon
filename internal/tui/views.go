package tui

import (
	"strings"

	"github.com/jesseduffield/gocui"
)

// contentText returns the text that should back the "content" view for the
// given tab, reading from the App's cached per-tab buffers.
func contentText(a *App, tab int) string {
	switch tab {
	case tabHeaders:
		return a.headersText
	case tabBody:
		return a.bodyText
	case tabAuth:
		return a.authText
	case tabParams:
		return a.paramsText
	case tabScripts:
		return a.scriptsText
	}
	return ""
}

// displayContentText returns the text loadContentTab (and the initial
// "content" view seed in layout) should render for the given tab: the raw
// per-tab buffer, except for a JSON body, which gets syntax-highlighted the
// same way live edits do in jsonBody's Edit handling (see editor.go).
func displayContentText(a *App, tab int) string {
	text := contentText(a, tab)
	if tab == tabBody && a.bodyType == 1 {
		return highlightJSON(text)
	}
	return text
}

// loadContentTab overwrites the "content" view with the text for the
// currently active tab. Call this only right after switching tabs (never
// while the user might be mid-edit on the same tab).
func loadContentTab(g *gocui.Gui, a *App) {
	v, err := g.View("content")
	if err != nil {
		return
	}
	v.Editable = true
	setViewText(v, displayContentText(a, a.activeTab))
	v.SetCursor(0, 0)
	v.SetOrigin(0, 0)
}

// syncFromViews pulls live-edited text out of the editable gocui views
// (url/content) and back into the App fields that buildURL/buildHeaders/
// buildBody/saveSession read from. Editable views are the source of truth
// for their text while focused, so anything reading App state must sync
// first.
func syncFromViews(g *gocui.Gui, a *App) {
	if v, err := g.View("url"); err == nil {
		a.urlValue = trimTrailingNewline(v.Buffer())
	}
	v, err := g.View("content")
	if err != nil {
		return
	}
	text := trimTrailingNewline(v.Buffer())
	switch a.activeTab {
	case tabHeaders:
		a.headersText = text
	case tabBody:
		a.bodyText = text
	case tabAuth:
		a.authText = text
	case tabParams:
		a.paramsText = text
	case tabScripts:
		a.scriptsText = text
	}
}

// trimTrailingNewline strips gocui.View.Buffer()'s always-present trailing
// newline, and the trailing space every line picks up as a side effect of
// how gocui's default editor inserts characters (View.writeRune leaves a
// dangling null cell after the last character typed on any line, which
// Buffer() renders as a literal space — see gocui's edit.go). Without this,
// every line ever typed into (URL, headers, body, ...) would silently gain
// a trailing space, which is invisible in the UI but breaks exact-match
// sends (URLs, JSON bodies, GraphQL queries).
func trimTrailingNewline(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	s = strings.Join(lines, "\n")
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}

// layout is gocui's per-iteration layout callback: it creates views on
// first use and resizes them on every call. It must never overwrite the
// buffer of an editable view that already exists (url/content/prompt) —
// only initial creation seeds their text.
func layout(g *gocui.Gui, a *App) error {
	maxX, maxY := g.Size()

	sidebarW := 26
	if sidebarW > maxX/3 {
		sidebarW = maxX / 3
	}
	if sidebarW < 10 {
		sidebarW = 10
	}
	rightX0 := sidebarW + 1
	if rightX0 >= maxX-2 {
		rightX0 = maxX - 3
	}

	// The method box + URL bar occupy a full-width row across the top; the
	// sidebar and the request/response columns start below it (y >= 3).
	methodW := 9
	if v, err := g.SetView("method", 0, 0, methodW, 2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Frame = false
		renderMethod(v, a)
	} else {
		renderMethod(v, a)
	}
	{
		focused := a.activePanel == panelURL && !a.subFocus
		mc := methodColor(methods[a.methodIndex])
		if focused {
			mc |= gocui.AttrBold
		}
		drawBorder(g, 0, 0, methodW, 2, mc, "")
	}

	if v, err := g.SetView("url", methodW+1, 0, maxX-1, 2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Frame = false
		v.Title = "URL"
		v.Editable = true
		setURLText(v, a.urlValue)
		if err := g.SetCurrentView("url"); err != nil {
			return err
		}
	}
	{
		focused := a.activePanel == panelURL && !a.subFocus
		drawBorder(g, methodW+1, 0, maxX-1, 2, borderColor(focused), "[^U] "+focusTitle("URL", focused))
	}

	if v, err := g.SetView("sidebar", 0, 3, sidebarW, maxY-3); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Frame = false
		renderSidebar(v, a)
	} else {
		renderSidebar(v, a)
	}
	if v, err := g.View("sidebar"); err == nil {
		focused := a.activePanel == panelSidebar
		drawBorder(g, 0, 3, sidebarW, maxY-3, borderColor(focused), "[^S] "+v.Title)
	}

	if v, err := g.SetView("tabs", rightX0, 3, maxX-1, 5); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Frame = false
		renderTabs(v, a)
	} else {
		renderTabs(v, a)
	}
	drawBorder(g, rightX0, 3, maxX-1, 5, borderColor(false), "")

	contentY1 := 6 + (maxY-3-6)/2
	if contentY1 < 10 {
		contentY1 = 10
	}
	if contentY1 > maxY-6 {
		contentY1 = maxY - 6
	}
	if contentY1 < 8 {
		contentY1 = 8
	}

	if v, err := g.SetView("content", rightX0, 6, maxX-1, contentY1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Frame = false
		v.Title = "Headers"
		v.Editable = true
		setViewText(v, displayContentText(a, a.activeTab))
	}
	if v, err := g.View("content"); err == nil {
		if a.activeTab == tabBody {
			v.Title = "Body (" + bodyTypes[a.bodyType] + ")"
		} else if a.activeTab == tabAuth {
			v.Title = "Auth (" + authTypes[a.authType] + ")"
		} else {
			v.Title = tabNames[a.activeTab]
		}
		drawBorder(g, rightX0, 6, maxX-1, contentY1, borderColor(a.subFocus), "[^B] "+focusTitle(v.Title, a.subFocus))
	}

	if a.contentFocusPending {
		a.contentFocusPending = false
		if err := g.SetCurrentView("content"); err != nil {
			return err
		}
	}

	if v, err := g.SetView("response", rightX0, contentY1+1, maxX-1, maxY-3); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Frame = false
		v.Wrap = true
		renderResponse(v, a)
	} else {
		v.Title = focusTitle("Response", a.activePanel == panelResponse)
	}
	{
		focused := a.activePanel == panelResponse
		drawBorder(g, rightX0, contentY1+1, maxX-1, maxY-3, borderColor(focused), "[^E] "+focusTitle("Response", focused))
	}

	if v, err := g.SetView("status", 0, maxY-2, maxX-1, maxY); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Frame = false
		renderStatus(v, a)
	} else {
		renderStatus(v, a)
	}

	if a.promptMode != "" {
		w, h := 50, 3
		x0, y0 := (maxX-w)/2, (maxY-h)/2
		if v, err := g.SetView("prompt", x0, y0, x0+w, y0+h); err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			// Unlike every other panel, the prompt floats on top of views
			// that were laid out (and painted) earlier this frame, so it
			// can't use the drawBorder-during-layout trick: those views
			// redraw their own full content after layout() returns and
			// would paint right over a manually-drawn border. Using
			// gocui's native Frame keeps the border/title tied to this
			// view's own draw call, which — since "prompt" is appended
			// last to g.views — always happens after (on top of)
			// everything underneath it.
			v.Frame = true
			v.Editable = a.promptMode != "confirmDelete"
			setViewText(v, a.promptText)
			renderPrompt(v, a)
			if err := g.SetCurrentView("prompt"); err != nil {
				return err
			}
		}
		// g.FgColor is read at draw time by gocui's own frame/title
		// rendering, which runs after layout() returns — so setting it
		// here (with no restore) is what actually colors the prompt's
		// border, unlike drawBorder's color param used elsewhere.
		g.FgColor = currentTheme.Primary
	} else {
		if _, err := g.View("prompt"); err == nil {
			_ = g.DeleteView("prompt")
			_ = g.SetCurrentView(panelSidebar)
		}
	}

	// The environment-variables editor is its own floating overlay — like
	// "prompt"/"codegen"/"help" — rather than reusing the request "content"
	// view the way it used to. Sharing that view made editing variables
	// look like just another request tab (Headers/Body/...), which is
	// exactly the ambiguity a dedicated overlay avoids.
	if a.envEditIdx >= 0 {
		w, h := 60, 16
		if w > maxX-4 {
			w = maxX - 4
		}
		if h > maxY-3 {
			h = maxY - 4
		}
		x0, y0 := (maxX-w)/2, (maxY-h)/2
		if v, err := g.SetView("envedit", x0, y0, x0+w, y0+h); err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			v.Frame = true
			v.Editable = true
			v.Title = "Environment: " + a.environments[a.envEditIdx].Name + " (Esc to save)"
			setViewText(v, a.envEditText)
			v.SetCursor(0, 0)
			v.SetOrigin(0, 0)
			if err := g.SetCurrentView("envedit"); err != nil {
				return err
			}
		}
		g.FgColor = currentTheme.Primary
	} else {
		if _, err := g.View("envedit"); err == nil {
			_ = g.DeleteView("envedit")
			_ = g.SetCurrentView(panelSidebar)
		}
	}

	if a.showCodegen {
		w, h := maxX-8, maxY-6
		if w < 20 {
			w = maxX
		}
		if h < 10 {
			h = maxY
		}
		x0, y0 := (maxX-w)/2, (maxY-h)/2
		v, err := g.SetView("codegen", x0, y0, x0+w, y0+h)
		if err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			v.Frame = true
			v.Editable = false
			if err := g.SetCurrentView("codegen"); err != nil {
				return err
			}
		}
		renderCodegen(v, a)
		g.FgColor = currentTheme.Primary
	} else {
		if _, err := g.View("codegen"); err == nil {
			_ = g.DeleteView("codegen")
			restoreView := a.activePanel
			if a.subFocus {
				restoreView = "content"
			}
			_ = g.SetCurrentView(restoreView)
		}
	}

	if a.fbMode != "" {
		w, h := maxX-8, maxY-6
		if w < 20 {
			w = maxX
		}
		if h < 6 {
			h = maxY
		}
		x0, y0 := (maxX-w)/2, (maxY-h)/2
		v, err := g.SetView("filebrowser", x0, y0, x0+w, y0+h)
		if err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			v.Frame = true
			v.Editable = false
			if err := g.SetCurrentView("filebrowser"); err != nil {
				return err
			}
		}
		renderFileBrowser(v, a)
		g.FgColor = currentTheme.Primary
	} else {
		if _, err := g.View("filebrowser"); err == nil {
			_ = g.DeleteView("filebrowser")
			// Don't yank focus back to the sidebar if a prompt has already
			// taken over this frame — export hands off to the exportPath
			// prompt the instant the browser closes.
			if a.promptMode == "" {
				_ = g.SetCurrentView(panelSidebar)
			}
		}
	}

	if a.showHelp {
		text, contentW, contentH := helpText(a.keymap)
		w, h := contentW+4, contentH+2
		if w > maxX-4 {
			w = maxX - 4
		}
		if h > maxY-3 {
			h = maxY - 4
		}
		x0, y0 := (maxX-w)/2, (maxY-h)/2
		if v, err := g.SetView("help", x0, y0, x0+w, y0+h); err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			// Same z-order trick as "prompt": appended last to g.views so
			// its own Frame/title draw call happens after (on top of)
			// everything laid out earlier this frame.
			v.Frame = true
			v.Title = "Keybindings (Esc or " + a.keymap.DisplayKey("toggleHelp") + " to close)"
			setViewText(v, text)
			if err := g.SetCurrentView("help"); err != nil {
				return err
			}
		}
		g.FgColor = currentTheme.Primary
	} else {
		if _, err := g.View("help"); err == nil {
			_ = g.DeleteView("help")
			restoreView := a.activePanel
			if a.subFocus {
				restoreView = "content"
			}
			_ = g.SetCurrentView(restoreView)
		}
	}

	// Show the terminal cursor only in views you actually type into (url,
	// content editor, prompt, envedit). Navigation panels like the sidebar
	// draw their own "> " selector, so a blinking block cursor there just
	// competes with it and reads as confusing. g.Cursor is global but read
	// once per frame against g.currentView (see gocui Gui.draw), so keying
	// it off the now-resolved current view's Editable flag is enough.
	if cv := g.CurrentView(); cv != nil {
		g.Cursor = cv.Editable
	} else {
		g.Cursor = false
	}

	return nil
}

func newLayoutFunc(a *App) gocui.Handler {
	return func(g *gocui.Gui) error {
		return layout(g, a)
	}
}
