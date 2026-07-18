package tui

import (
	"fmt"

	"github.com/jesseduffield/gocui"
)

// Theme holds the semantic color roles curlmoon paints with. gocui only
// supports termbox's basic 16-color palette (see ansiFG below), so "light"
// vs "dark" just means picking a different assignment from that same
// 16-color set — there's no true RGB theming available here.
type Theme struct {
	Primary   gocui.Attribute
	Secondary gocui.Attribute
	Success   gocui.Attribute
	Error     gocui.Attribute
	Muted     gocui.Attribute
	Magenta   gocui.Attribute
}

var darkTheme = Theme{
	Primary:   gocui.ColorCyan,
	Secondary: gocui.ColorYellow,
	Success:   gocui.ColorGreen,
	Error:     gocui.ColorRed,
	Muted:     gocui.ColorWhite,
	Magenta:   gocui.ColorMagenta,
}

// lightTheme swaps the "muted" role from white to black, since white text
// on a light terminal background is nearly invisible; the rest of the
// 16-color palette reads fine on either background.
var lightTheme = Theme{
	Primary:   gocui.ColorBlue,
	Secondary: gocui.ColorYellow,
	Success:   gocui.ColorGreen,
	Error:     gocui.ColorRed,
	Muted:     gocui.ColorBlack,
	Magenta:   gocui.ColorMagenta,
}

// currentTheme is the active color set, swapped by App.ToggleTheme.
var currentTheme = darkTheme

// themeByName resolves a persisted theme name ("dark"/"light") to a Theme,
// defaulting to dark for anything else (including the empty string).
func themeByName(name string) Theme {
	if name == "light" {
		return lightTheme
	}
	return darkTheme
}

func themeName(t Theme) string {
	if t == lightTheme {
		return "light"
	}
	return "dark"
}

// ansiFG maps a gocui.Attribute color to its SGR foreground code.
func ansiFG(c gocui.Attribute) int {
	switch c &^ (gocui.AttrBold | gocui.AttrUnderline | gocui.AttrReverse) {
	case gocui.ColorBlack:
		return 30
	case gocui.ColorRed:
		return 31
	case gocui.ColorGreen:
		return 32
	case gocui.ColorYellow:
		return 33
	case gocui.ColorBlue:
		return 34
	case gocui.ColorMagenta:
		return 35
	case gocui.ColorCyan:
		return 36
	case gocui.ColorWhite:
		return 37
	default:
		return 39
	}
}

// ansiWrap colors text using an SGR escape sequence gocui's Views understand
// when the text is written into a View's buffer.
func ansiWrap(text string, fg gocui.Attribute, bold bool) string {
	if bold {
		return fmt.Sprintf("\x1b[1;%dm%s\x1b[0m", ansiFG(fg), text)
	}
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", ansiFG(fg), text)
}

// methodColor returns the color associated with an HTTP method badge.
func methodColor(method string) gocui.Attribute {
	switch method {
	case "GET":
		return currentTheme.Success
	case "POST":
		return currentTheme.Primary
	case "PUT":
		return currentTheme.Secondary
	case "DELETE":
		return currentTheme.Error
	case "PATCH":
		return currentTheme.Magenta
	default:
		return currentTheme.Success
	}
}

// statusColor returns the color associated with an HTTP response status code.
func statusColor(statusCode int) gocui.Attribute {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return currentTheme.Success
	case statusCode >= 400:
		return currentTheme.Error
	default:
		return currentTheme.Secondary
	}
}
