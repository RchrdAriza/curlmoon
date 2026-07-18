package tui

import "github.com/jesseduffield/gocui"

// borderColor returns the border/title color for a panel given whether it
// currently has focus — the visual cue lazygit uses to show which panel
// keystrokes go to.
func borderColor(focused bool) gocui.Attribute {
	if focused {
		return currentTheme.Primary
	}
	return currentTheme.Muted
}

// drawBorder draws a view's frame, corners and title manually. gocui's own
// frame drawing (View.Frame = true) paints every view's border with the same
// global g.FgColor, so there is no way to color one focused panel's border
// differently from the rest through the public API. Views are instead kept
// at Frame = false and their border is painted here, once per layout tick,
// with whatever color the caller computes for that panel's focus state.
func drawBorder(g *gocui.Gui, x0, y0, x1, y1 int, color gocui.Attribute, title string) {
	maxX, maxY := g.Size()
	prev := g.FgColor
	g.FgColor = color
	defer func() { g.FgColor = prev }()

	for x := x0 + 1; x < x1 && x < maxX; x++ {
		if x < 0 {
			continue
		}
		if y0 > -1 && y0 < maxY {
			_ = g.SetRune(x, y0, '─')
		}
		if y1 > -1 && y1 < maxY {
			_ = g.SetRune(x, y1, '─')
		}
	}
	for y := y0 + 1; y < y1 && y < maxY; y++ {
		if y < 0 {
			continue
		}
		if x0 > -1 && x0 < maxX {
			_ = g.SetRune(x0, y, '│')
		}
		if x1 > -1 && x1 < maxX {
			_ = g.SetRune(x1, y, '│')
		}
	}
	if x0 >= 0 && y0 >= 0 && x0 < maxX && y0 < maxY {
		_ = g.SetRune(x0, y0, '┌')
	}
	if x1 >= 0 && y0 >= 0 && x1 < maxX && y0 < maxY {
		_ = g.SetRune(x1, y0, '┐')
	}
	if x0 >= 0 && y1 >= 0 && x0 < maxX && y1 < maxY {
		_ = g.SetRune(x0, y1, '└')
	}
	if x1 >= 0 && y1 >= 0 && x1 < maxX && y1 < maxY {
		_ = g.SetRune(x1, y1, '┘')
	}

	if title != "" && y0 >= 0 && y0 < maxY {
		for i, ch := range title {
			x := x0 + i + 2
			if x < 0 {
				continue
			}
			if x > x1-2 || x >= maxX {
				break
			}
			_ = g.SetRune(x, y0, ch)
		}
	}
}
