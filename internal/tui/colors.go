package tui

import (
	"fmt"

	"github.com/jesseduffield/gocui"
)

// Color roles, mapped onto gocui's basic 16-color palette (the only palette
// its ANSI interpreter supports — see escape.go).
const (
	colorPrimary   = gocui.ColorCyan
	colorSecondary = gocui.ColorYellow
	colorSuccess   = gocui.ColorGreen
	colorError     = gocui.ColorRed
	colorMuted     = gocui.ColorWhite
	colorMagenta   = gocui.ColorMagenta
)

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
		return colorSuccess
	case "POST":
		return colorPrimary
	case "PUT":
		return colorSecondary
	case "DELETE":
		return colorError
	case "PATCH":
		return colorMagenta
	default:
		return colorSuccess
	}
}

// statusColor returns the color associated with an HTTP response status code.
func statusColor(statusCode int) gocui.Attribute {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return colorSuccess
	case statusCode >= 400:
		return colorError
	default:
		return colorSecondary
	}
}
