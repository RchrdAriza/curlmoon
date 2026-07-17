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
	if a.promptMode != "" {
		return "Enter confirm  ·  Esc cancel"
	}
	if a.subFocus {
		return "Esc save & exit editor  ·  Ctrl+R send  ·  Tab next panel"
	}
	switch a.activePanel {
	case panelSidebar:
		return "↑↓/jk navigate  ·  Enter open/toggle  ·  n new  ·  a add request  ·  r rename  ·  d delete  ·  v edit vars  ·  Tab next panel  ·  q quit"
	case panelURL:
		return "↑↓ change method  ·  ←→ switch tab  ·  Enter edit content  ·  Ctrl+R send  ·  Tab next panel  ·  Alt+1-4 jump panel"
	case panelResponse:
		return "↑↓ scroll  ·  PgUp/PgDn page  ·  Tab next panel"
	}
	return "Tab cycle panels  ·  Alt+1-4 jump panel  ·  q quit"
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
