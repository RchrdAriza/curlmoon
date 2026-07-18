package tui

import (
	"strings"

	"curlmoon/internal/codegen"
	"github.com/jesseduffield/gocui"
)

// renderCodegen fills the "codegen" overlay with the current request
// rendered in the active language, plus a language switcher strip.
func renderCodegen(v *gocui.View, a *App) {
	var langs []string
	for i, l := range codegen.Langs {
		name := " " + l.String() + " "
		if i == a.codegenLang {
			langs = append(langs, ansiWrap(name, currentTheme.Primary, true))
		} else {
			langs = append(langs, ansiWrap(name, currentTheme.Muted, false))
		}
	}
	v.Title = "Generate code (Ctrl+G close, Tab/Shift+Tab language)"

	var b strings.Builder
	b.WriteString(strings.Join(langs, " "))
	b.WriteString("\n\n")
	b.WriteString(a.codegenSnippet())
	setViewText(v, b.String())
	v.SetOrigin(0, 0)
}
