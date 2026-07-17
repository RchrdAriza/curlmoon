package tui

import (
	"github.com/jesseduffield/gocui"
)

const authPlaceholder = "Auth helpers coming soon.\n\nSupports: None, Basic, Bearer Token, API Key, OAuth 2.0"

// contentText returns the text that should back the "content" view for the
// given tab, reading from the App's cached per-tab buffers.
func contentText(a *App, tab int) string {
	switch tab {
	case tabHeaders:
		return a.headersText
	case tabBody:
		return a.bodyText
	case tabAuth:
		return authPlaceholder
	case tabParams:
		return a.paramsText
	}
	return ""
}

// loadContentTab overwrites the "content" view with the text for the
// currently active tab. Call this only right after switching tabs (never
// while the user might be mid-edit on the same tab).
func loadContentTab(g *gocui.Gui, a *App) {
	v, err := g.View("content")
	if err != nil {
		return
	}
	v.Editable = a.activeTab != tabAuth
	setViewText(v, contentText(a, a.activeTab))
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
	if v, err := g.View("content"); err == nil && a.activeTab != tabAuth {
		text := trimTrailingNewline(v.Buffer())
		switch a.activeTab {
		case tabHeaders:
			a.headersText = text
		case tabBody:
			a.bodyText = text
		case tabParams:
			a.paramsText = text
		}
	}
}

func trimTrailingNewline(s string) string {
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

	if v, err := g.SetView("sidebar", 0, 0, sidebarW, maxY-3); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Frame = true
		renderSidebar(v, a)
	} else {
		renderSidebar(v, a)
	}

	methodW := 9
	if v, err := g.SetView("method", rightX0, 0, rightX0+methodW, 2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Frame = true
		renderMethod(v, a)
	} else {
		renderMethod(v, a)
	}

	if v, err := g.SetView("url", rightX0+methodW+1, 0, maxX-1, 2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Frame = true
		v.Title = "URL"
		v.Editable = true
		setViewText(v, a.urlValue)
		if err := g.SetCurrentView("url"); err != nil {
			return err
		}
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
		v.Frame = true
		v.Title = "Headers"
		v.Editable = a.activeTab != tabAuth
		setViewText(v, contentText(a, a.activeTab))
	}
	if v, err := g.View("content"); err == nil {
		v.Title = tabNames[a.activeTab]
	}

	if v, err := g.SetView("response", rightX0, contentY1+1, maxX-1, maxY-3); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Frame = true
		renderResponse(v, a)
	} else {
		v.Title = focusTitle("Response", a.activePanel == panelResponse)
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
			v.Frame = true
			v.Editable = a.promptMode != "confirmDelete"
			setViewText(v, a.promptText)
			renderPrompt(v, a)
			if err := g.SetCurrentView("prompt"); err != nil {
				return err
			}
		}
	} else {
		if _, err := g.View("prompt"); err == nil {
			_ = g.DeleteView("prompt")
			_ = g.SetCurrentView(panelSidebar)
		}
	}

	return nil
}

func newLayoutFunc(a *App) gocui.Handler {
	return func(g *gocui.Gui) error {
		return layout(g, a)
	}
}
