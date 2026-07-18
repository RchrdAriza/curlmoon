package tui

import "testing"

func TestIsBracketCloser(t *testing.T) {
	for _, ch := range []rune{'}', ']', ')'} {
		if !isBracketCloser(ch) {
			t.Errorf("isBracketCloser(%q) = false, want true", ch)
		}
	}
	for _, ch := range []rune{'{', '[', '(', 'a', '"', 0} {
		if isBracketCloser(ch) {
			t.Errorf("isBracketCloser(%q) = true, want false", ch)
		}
	}
}

func TestBracketClosers(t *testing.T) {
	want := map[rune]rune{'{': '}', '[': ']', '(': ')'}
	for open, close := range want {
		if got := bracketClosers[open]; got != close {
			t.Errorf("bracketClosers[%q] = %q, want %q", open, got, close)
		}
	}
}
