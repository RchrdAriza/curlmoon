package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/jesseduffield/gocui"
)

// setViewText replaces a view's whole content. Only ever call this for
// views whose content is fully derived from App state (never for a view the
// user is actively typing into, or their keystrokes would be clobbered).
func setViewText(v *gocui.View, text string) {
	v.Clear()
	fmt.Fprint(v, text)
}

// setURLText replaces the URL view's content and parks the cursor at the
// end of it (scrolling the origin so it's visible), matching where you'd
// want to keep typing after loading a request — gocui otherwise leaves the
// cursor at (0,0) after a Clear+Write, which put every fresh keystroke at
// the start of the URL instead of the end.
func setURLText(v *gocui.View, text string) {
	setViewText(v, text)
	n := len([]rune(text))
	maxX, _ := v.Size()
	if maxX < 1 {
		maxX = 1
	}
	if n < maxX {
		_ = v.SetOrigin(0, 0)
		_ = v.SetCursor(n, 0)
	} else {
		_ = v.SetOrigin(n-maxX+1, 0)
		_ = v.SetCursor(maxX-1, 0)
	}
}

func focusTitle(name string, focused bool) string {
	if focused {
		return "▸ " + name
	}
	return " " + name
}

func renderSidebar(v *gocui.View, a *App) {
	v.Title = focusTitle("Collections", a.activePanel == panelSidebar)

	_, height := v.Size()
	maxItems := height
	if maxItems < 1 {
		maxItems = 1
	}

	var b strings.Builder
	visible := a.sidebar
	if a.sidebarOff > 0 && a.sidebarOff < len(visible) {
		visible = visible[a.sidebarOff:]
	}
	for i, item := range visible {
		if i >= maxItems {
			break
		}
		idx := a.sidebarOff + i
		prefix := ""
		if item.indent > 0 {
			prefix = strings.Repeat("  ", item.indent)
		}
		marker := "  "
		if idx == a.sidebarSel {
			marker = "> "
		}
		if item.isFolder {
			icon := "[-]"
			if a.collapsed[sidebarFolderKey(item)] {
				icon = "[+]"
			}
			line := marker + prefix + icon + " " + item.name
			b.WriteString(ansiWrap(line, colorPrimary, true))
		} else {
			meth := ansiWrap(fmt.Sprintf("%-6s", item.method), methodColor(item.method), true)
			display := item.name
			if len(display) > 20 {
				display = display[:18] + ".."
			}
			b.WriteString(fmt.Sprintf("%s%s%s %s", marker, prefix, meth, display))
		}
		b.WriteString("\n")
	}
	setViewText(v, b.String())
}

func renderMethod(v *gocui.View, a *App) {
	setViewText(v, ansiWrap(methods[a.methodIndex], methodColor(methods[a.methodIndex]), true))
}

func renderTabs(v *gocui.View, a *App) {
	var parts []string
	for i, name := range tabNames {
		if i == a.activeTab {
			parts = append(parts, ansiWrap(" "+name+" ", colorPrimary, true))
		} else {
			parts = append(parts, ansiWrap(" "+name+" ", colorMuted, false))
		}
	}
	setViewText(v, strings.Join(parts, " "))
}

// footerHints returns the always-visible keybinding help for whatever has
// focus right now, so users never have to guess (e.g. that ↑/↓ cycle the
// HTTP method while the URL panel is focused).
func footerHints(a *App) string {
	if a.showHelp {
		return "Esc or Ctrl+/ close help"
	}
	if a.promptMode != "" {
		return "Enter confirm  ·  Esc cancel"
	}
	if a.subFocus {
		return "Esc save & exit editor  ·  Ctrl+R send  ·  Tab next panel  ·  Ctrl+/ help"
	}
	switch a.activePanel {
	case panelSidebar:
		return "↑↓/jk navigate  ·  Enter open/toggle  ·  n new  ·  a add request  ·  r rename  ·  d delete  ·  v edit vars  ·  Tab next panel  ·  Ctrl+/ help  ·  q quit"
	case panelURL:
		return "←→/Home/End move in URL  ·  ↑↓/Ctrl+K,J method  ·  Ctrl+P,N switch tab  ·  Enter edit content  ·  Ctrl+R send  ·  Tab next panel  ·  Ctrl+S,U,B,E jump panel  ·  Ctrl+/ help"
	case panelResponse:
		return "↑↓ scroll  ·  PgUp/PgDn page  ·  Tab next panel  ·  Ctrl+/ help"
	}
	return "Tab cycle panels  ·  Ctrl+S,U,B,E jump panel  ·  Ctrl+/ help  ·  q quit"
}

func renderStatus(v *gocui.View, a *App) {
	text := a.statusMsg
	if a.sending {
		text = "⏳ " + text
	}
	hints := ansiWrap(footerHints(a), colorMuted, false)
	setViewText(v, hints+"\n"+text)
}

func renderResponse(v *gocui.View, a *App) {
	v.Title = focusTitle("Response", a.activePanel == panelResponse)

	if !a.showResp && a.respErr == nil {
		setViewText(v, "Send a request to see the response here\n\nTry: Ctrl+R")
		return
	}
	if a.respErr != nil {
		setViewText(v, ansiWrap("Error:", colorError, true)+"\n"+fmt.Sprintf("%v", a.respErr))
		return
	}
	if a.response == nil {
		setViewText(v, "")
		return
	}

	statusText := ansiWrap(fmt.Sprintf("%d %s", a.response.StatusCode, a.response.Status), statusColor(a.response.StatusCode), true)
	infoText := ansiWrap(fmt.Sprintf("%s  |  %d bytes", a.response.Elapsed.Round(time.Millisecond), a.response.Size), colorMuted, false)

	var headerPreview strings.Builder
	headerLines := strings.Split(a.response.HeaderStr, "\n")
	maxHeaderLines := 4
	for i, line := range headerLines {
		if i >= maxHeaderLines {
			headerPreview.WriteString("...\n")
			break
		}
		if line == "" {
			continue
		}
		key, val, ok := strings.Cut(line, ": ")
		if ok {
			headerPreview.WriteString(ansiWrap(key, colorPrimary, true) + ": " + val + "\n")
		} else {
			headerPreview.WriteString(line + "\n")
		}
	}

	body := highlightJSON(a.response.Body)

	setViewText(v, statusText+"  "+infoText+"\n"+headerPreview.String()+"\n"+body)
}

// helpSections lists every keybinding grouped by the context it applies in,
// rendered by the Ctrl+/ overlay (see toggleHelp in keybindings.go).
var helpSections = []struct {
	title string
	rows  [][2]string
}{
	{"Global", [][2]string{
		{"Tab", "cycle sidebar → URL → response"},
		{"Ctrl+S / Ctrl+U / Ctrl+E", "jump to sidebar / URL / response"},
		{"Ctrl+B", "jump into content editor"},
		{"Ctrl+/", "toggle this help"},
		{"q, Ctrl+C", "quit"},
	}},
	{"Sidebar", [][2]string{
		{"↑↓, j/k", "navigate"},
		{"Enter", "open request / toggle folder"},
		{"n", "new collection / environment"},
		{"a", "add request"},
		{"r", "rename"},
		{"d", "delete"},
		{"v", "edit environment vars"},
	}},
	{"URL bar", [][2]string{
		{"←→, Home/End", "move cursor"},
		{"↑↓, Ctrl+K/J", "cycle HTTP method"},
		{"Ctrl+P/N", "switch tab (Headers/Body/Auth/Params)"},
		{"Enter", "edit tab content"},
		{"Ctrl+R", "send request"},
	}},
	{"Content editor", [][2]string{
		{"Esc", "save & exit editor"},
		{"Ctrl+R", "send request"},
	}},
	{"Response", [][2]string{
		{"↑↓", "scroll"},
		{"PgUp/PgDn", "page"},
	}},
	{"Prompt", [][2]string{
		{"Enter", "confirm"},
		{"Esc", "cancel"},
		{"y/n", "confirm/cancel delete"},
	}},
}

// helpText renders helpSections into the fixed-width columns the overlay
// draws, and reports the content's natural width/height so the caller can
// size the floating view around it.
func helpText() (text string, width, height int) {
	keyW := 0
	for _, s := range helpSections {
		for _, row := range s.rows {
			if len(row[0]) > keyW {
				keyW = len(row[0])
			}
		}
	}
	var b strings.Builder
	lineW := 0
	lines := 0
	for i, s := range helpSections {
		if i > 0 {
			b.WriteString("\n")
			lines++
		}
		b.WriteString(ansiWrap(s.title, colorSecondary, true))
		b.WriteString("\n")
		lines++
		for _, row := range s.rows {
			line := fmt.Sprintf("  %-*s  %s", keyW, row[0], row[1])
			if len(line) > lineW {
				lineW = len(line)
			}
			b.WriteString(ansiWrap(fmt.Sprintf("  %-*s", keyW, row[0]), colorPrimary, true))
			b.WriteString("  " + row[1] + "\n")
			lines++
		}
	}
	return b.String(), lineW, lines
}

func renderPrompt(v *gocui.View, a *App) {
	switch a.promptMode {
	case "newCollection":
		v.Title = "New collection name"
	case "newRequest":
		v.Title = "Save as (request name)"
	case "rename":
		v.Title = "Rename to"
	case "confirmDelete":
		v.Title = fmt.Sprintf("Delete %q? (y/n)", a.promptTarget.name)
	case "newEnvironment":
		v.Title = "New environment name"
	case "renameEnv":
		v.Title = "Rename environment to"
	case "confirmDeleteEnv":
		v.Title = fmt.Sprintf("Delete environment %q? (y/n)", a.promptTarget.name)
	}
}
