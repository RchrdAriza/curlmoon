package tui

import (
	"fmt"
	"strings"
	"time"

	"curlmoon/internal/config"
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
// the start of the URL instead of the end. text is highlighted for
// {{variable}} tokens before writing; the cursor math still uses the plain
// rune count since ansiWrap's escapes are consumed by gocui without
// emitting cells (see recolorJSON in editor.go).
func setURLText(v *gocui.View, text string) {
	setViewText(v, highlightURL(text))
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
			b.WriteString(ansiWrap(line, currentTheme.Primary, true))
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
			parts = append(parts, ansiWrap(" "+name+" ", currentTheme.Primary, true))
		} else {
			parts = append(parts, ansiWrap(" "+name+" ", currentTheme.Muted, false))
		}
	}
	setViewText(v, strings.Join(parts, " "))
}

// footerHints returns the always-visible keybinding help for whatever has
// focus right now, so users never have to guess (e.g. that ↑/↓ cycle the
// HTTP method while the URL panel is focused).
func footerHints(a *App) string {
	if a.fbMode == "import" {
		return "↑↓ nav · Enter open/import · ← up · Esc cancel"
	}
	if a.fbMode == "export" {
		return "↑↓ nav · Enter open · ← up · Ctrl+S save · Esc cancel"
	}
	if a.showCodegen {
		return "Tab language · Ctrl+G close"
	}
	if a.showHelp {
		return "Esc close help"
	}
	if a.promptMode != "" {
		return "Enter confirm · Esc cancel"
	}
	if a.subFocus {
		return "Esc exit editor · Ctrl+R send · Tab panel · Ctrl+/ help"
	}
	switch a.activePanel {
	case panelSidebar:
		return "↑↓ nav · Enter open · n new · a add · r rename · d delete · v vars · x export · i import · Ctrl+/ help"
	case panelURL:
		hint := "←→ move · ↑↓ method · Ctrl+P,N tab · Enter edit"
		if a.activeTab == tabBody {
			hint += " · Ctrl+Y body type"
		}
		if a.activeTab == tabAuth {
			hint += " · Ctrl+Y auth type"
		}
		return hint + " · Ctrl+R send · Ctrl+G code · Ctrl+/ help"
	case panelResponse:
		return "↑↓ scroll · PgUp/PgDn page · Tab panel · Ctrl+/ help"
	}
	return "Tab panels · Ctrl+/ help · q quit"
}

// renderStatus draws the single-line footer: just the context hints. It no
// longer echoes a.statusMsg — that line duplicated data already shown in the
// dedicated panels (response status, loaded method/URL, body/auth type). A
// spinner glyph is prepended while a request is in flight so there's still
// feedback that something is happening.
func renderStatus(v *gocui.View, a *App) {
	hints := footerHints(a)
	if a.sending {
		hints = "⏳ Sending…  ·  " + hints
	}
	setViewText(v, ansiWrap(hints, currentTheme.Muted, false))
}

func renderResponse(v *gocui.View, a *App) {
	v.Title = focusTitle("Response", a.activePanel == panelResponse)

	if !a.showResp && a.respErr == nil {
		setViewText(v, "Send a request to see the response here\n\nTry: Ctrl+R")
		return
	}
	if a.respErr != nil {
		setViewText(v, ansiWrap("Error:", currentTheme.Error, true)+"\n"+fmt.Sprintf("%v", a.respErr))
		return
	}
	if a.response == nil {
		setViewText(v, "")
		return
	}

	statusText := ansiWrap(fmt.Sprintf("%d %s", a.response.StatusCode, a.response.Status), statusColor(a.response.StatusCode), true)
	infoText := ansiWrap(fmt.Sprintf("%s  |  %d bytes", a.response.Elapsed.Round(time.Millisecond), a.response.Size), currentTheme.Muted, false)

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
			headerPreview.WriteString(ansiWrap(key, currentTheme.Primary, true) + ": " + val + "\n")
		} else {
			headerPreview.WriteString(line + "\n")
		}
	}

	body := highlightJSON(prettyJSON(a.response.Body))

	testSummary := renderTestSummary(a)

	setViewText(v, statusText+"  "+infoText+"\n"+headerPreview.String()+testSummary+"\n"+body)
}

// renderTestSummary renders the Scripts tab's test results (if any test
// script ran) as a pass/fail line plus per-test detail for failures.
func renderTestSummary(a *App) string {
	var b strings.Builder
	if a.scriptErr != "" {
		b.WriteString(ansiWrap("Script error: ", currentTheme.Error, true) + a.scriptErr + "\n")
	}
	if len(a.testResults) == 0 {
		return b.String()
	}
	passed, failed := 0, 0
	for _, tr := range a.testResults {
		if tr.Passed {
			passed++
		} else {
			failed++
		}
	}
	summary := fmt.Sprintf("✓ %d passed", passed)
	if failed > 0 {
		summary += fmt.Sprintf(" · ✗ %d failed", failed)
	}
	color := currentTheme.Success
	if failed > 0 {
		color = currentTheme.Error
	}
	b.WriteString(ansiWrap(summary, color, true) + "\n")
	for _, tr := range a.testResults {
		if tr.Passed {
			continue
		}
		b.WriteString(ansiWrap("  ✗ "+tr.Name, currentTheme.Error, false))
		if tr.Err != "" {
			b.WriteString(": " + tr.Err)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// helpSections lists every keybinding grouped by the context it applies in,
// rendered by the Ctrl+/ overlay (see toggleHelp in keybindings.go). Key
// labels are pulled from km so an edited ~/.curlmoon/keybindings.json shows
// up here too, instead of drifting from the actual bindings.
func helpSections(km config.Keymap) []struct {
	title string
	rows  [][2]string
} {
	k := km.DisplayKey
	return []struct {
		title string
		rows  [][2]string
	}{
		{"Global", [][2]string{
			{k("cycleFocus"), "cycle sidebar → URL → response"},
			{k("jumpSidebar") + "/" + k("jumpURL") + "/" + k("jumpResponse"), "jump to sidebar / URL / response"},
			{k("jumpContent"), "jump into content editor"},
			{k("toggleHelp"), "toggle this help"},
			{k("toggleCodegen"), "generate code (curl/Go/Python/JS)"},
			{k("toggleTheme"), "toggle light/dark theme"},
			{k("quit"), "quit (sidebar/response/overlays only, not while typing)"},
			{"Ctrl+C", "force quit, always works"},
		}},
		{"Sidebar", [][2]string{
			{k("sidebarUp") + "/" + k("sidebarDown") + ", j/k", "navigate"},
			{k("sidebarEnter"), "open request / toggle folder"},
			{k("sidebarNewCollection"), "new collection / environment"},
			{k("sidebarNewRequest"), "add request"},
			{k("sidebarRename"), "rename"},
			{k("sidebarDelete"), "delete"},
			{k("sidebarEditVars"), "edit environment vars"},
			{k("sidebarExport"), "export collection"},
			{k("sidebarImport"), "import collection"},
		}},
		{"URL bar", [][2]string{
			{"←→, Home/End", "move cursor"},
			{k("urlMethodUp") + "/" + k("urlMethodDown") + ", Ctrl+K/J", "cycle HTTP method"},
			{k("urlSwitchTabPrev") + "/" + k("urlSwitchTabNext"), "switch tab (Headers/Body/Auth/Params/Scripts)"},
			{k("cycleBodyType"), "cycle body type (Body tab) / auth type (Auth tab)"},
			{k("urlEnterContent"), "edit tab content"},
			{k("sendRequest"), "send request"},
		}},
		{"Content editor", [][2]string{
			{k("contentEsc"), "save & exit editor"},
			{k("sendRequest"), "send request"},
		}},
		{"Response", [][2]string{
			{k("responseScrollUp") + "/" + k("responseScrollDown"), "scroll"},
			{k("responsePageUp") + "/" + k("responsePageDown"), "page"},
		}},
		{"Code gen overlay", [][2]string{
			{"Tab/Backspace", "cycle language"},
			{k("toggleCodegen"), "close"},
		}},
		{"Prompt", [][2]string{
			{"Enter", "confirm"},
			{"Esc", "cancel"},
			{"y/n", "confirm/cancel delete"},
		}},
	}
}

// helpText renders helpSections(km) into the fixed-width columns the
// overlay draws, and reports the content's natural width/height so the
// caller can size the floating view around it.
func helpText(km config.Keymap) (text string, width, height int) {
	sections := helpSections(km)
	keyW := 0
	for _, s := range sections {
		for _, row := range s.rows {
			if len(row[0]) > keyW {
				keyW = len(row[0])
			}
		}
	}
	var b strings.Builder
	lineW := 0
	lines := 0
	for i, s := range sections {
		if i > 0 {
			b.WriteString("\n")
			lines++
		}
		b.WriteString(ansiWrap(s.title, currentTheme.Secondary, true))
		b.WriteString("\n")
		lines++
		for _, row := range s.rows {
			line := fmt.Sprintf("  %-*s  %s", keyW, row[0], row[1])
			if len(line) > lineW {
				lineW = len(line)
			}
			b.WriteString(ansiWrap(fmt.Sprintf("  %-*s", keyW, row[0]), currentTheme.Primary, true))
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
	case "exportPath":
		v.Title = "Export to path"
	case "importPath":
		v.Title = "Import from path"
	}
}
