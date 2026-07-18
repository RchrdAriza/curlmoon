package tui

import "testing"

// TestTabAtX pins down the column→tab mapping tabsTap relies on. The tab row
// renders as " <name> " per tab (len(name)+2 cols) joined by one space, so
// with tabNames = Headers/Body/Auth/Params/Scripts the ranges are:
//
//	Headers 0-8, sep 9, Body 10-15, sep 16, Auth 17-22, sep 23,
//	Params 24-31, sep 32, Scripts 33-41.
func TestTabAtX(t *testing.T) {
	cases := []struct {
		cx   int
		want int
	}{
		{0, tabHeaders},
		{8, tabHeaders},
		{9, -1}, // separator gap
		{10, tabBody},
		{15, tabBody},
		{17, tabAuth},
		{24, tabParams},
		{33, tabScripts},
		{41, tabScripts},
		{42, -1}, // past the last tab
	}
	for _, c := range cases {
		if got := tabAtX(c.cx); got != c.want {
			t.Errorf("tabAtX(%d) = %d, want %d", c.cx, got, c.want)
		}
	}
}
