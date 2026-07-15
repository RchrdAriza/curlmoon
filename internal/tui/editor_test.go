package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewKeyValueEditor(t *testing.T) {
	e := NewKeyValueEditor("Key", "Value")
	if e == nil {
		t.Fatal("expected non-nil editor")
	}
	if e.KeyLabel != "Key" {
		t.Errorf("expected KeyLabel=Key, got %s", e.KeyLabel)
	}
	if e.ValLabel != "Value" {
		t.Errorf("expected ValLabel=Value, got %s", e.ValLabel)
	}
	if len(e.Rows) != 1 {
		t.Errorf("expected 1 row, got %d", len(e.Rows))
	}
	if e.FocusIdx != -1 {
		t.Errorf("expected FocusIdx=-1, got %d", e.FocusIdx)
	}
}

func TestEditorFocusBlur(t *testing.T) {
	e := NewKeyValueEditor("K", "V")
	if e.Focused() {
		t.Error("expected not focused initially")
	}
	e.Focus()
	if !e.Focused() {
		t.Error("expected focused after Focus()")
	}
	if e.FocusIdx != 0 {
		t.Errorf("expected FocusIdx=0 after focus, got %d", e.FocusIdx)
	}
	e.Blur()
	if e.Focused() {
		t.Error("expected not focused after Blur()")
	}
	if e.FocusIdx != -1 {
		t.Errorf("expected FocusIdx=-1 after blur, got %d", e.FocusIdx)
	}
}

func TestEditorTabCycling(t *testing.T) {
	e := NewKeyValueEditor("K", "V")
	e.Focus()
	e.addEmptyRow()

	initialFocus := e.FocusIdx

	// Tab should cycle forward
	e.HandleKey(tea.KeyMsg{Type: tea.KeyTab})
	if e.FocusIdx == initialFocus {
		t.Error("expected FocusIdx to change after Tab")
	}

	// Shift+Tab should cycle backward
	prev := e.FocusIdx
	e.HandleKey(tea.KeyMsg{Type: tea.KeyShiftTab})
	if e.FocusIdx == prev {
		t.Error("expected FocusIdx to change after Shift+Tab")
	}
}

func TestEditorAddRow(t *testing.T) {
	e := NewKeyValueEditor("K", "V")
	e.Focus()

	// Navigate to add button (last focusable)
	for e.FocusIdx != e.totalFocusable()-1 {
		e.HandleKey(tea.KeyMsg{Type: tea.KeyTab})
	}

	// Press enter on add button
	e.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if len(e.Rows) != 2 {
		t.Errorf("expected 2 rows after adding, got %d", len(e.Rows))
	}
}

func TestEditorRemoveRow(t *testing.T) {
	e := NewKeyValueEditor("K", "V")
	e.addEmptyRow()
	e.Focus()

	if len(e.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(e.Rows))
	}

	// Focus the second row's key field
	e.FocusIdx = 2
	e.syncFocus()

	// Backspace on empty key should remove row
	e.HandleKey(tea.KeyMsg{Type: tea.KeyBackspace})
	if len(e.Rows) != 1 {
		t.Errorf("expected 1 row after removal, got %d", len(e.Rows))
	}
}

func TestEditorPairs(t *testing.T) {
	e := NewKeyValueEditor("K", "V")
	e.Rows[0].key.SetValue("Content-Type")
	e.Rows[0].value.SetValue("application/json")
	e.addEmptyRow()
	e.Rows[1].key.SetValue("Authorization")
	e.Rows[1].value.SetValue("Bearer token")

	pairs := e.Pairs()
	if len(pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(pairs))
	}
	if pairs[0].Key != "Content-Type" {
		t.Errorf("expected Content-Type, got %s", pairs[0].Key)
	}
	if pairs[0].Value != "application/json" {
		t.Errorf("expected application/json, got %s", pairs[0].Value)
	}
	if pairs[1].Key != "Authorization" {
		t.Errorf("expected Authorization, got %s", pairs[1].Key)
	}
}

func TestEditorToMap(t *testing.T) {
	e := NewKeyValueEditor("K", "V")
	e.Rows[0].key.SetValue("Accept")
	e.Rows[0].value.SetValue("*/*")

	m := e.ToMap()
	if len(m) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(m))
	}
	if m["Accept"] != "*/*" {
		t.Errorf("expected */*, got %s", m["Accept"])
	}
}

func TestEditorEmptyRowsSkipped(t *testing.T) {
	e := NewKeyValueEditor("K", "V")
	e.Rows[0].key.SetValue("")
	e.Rows[0].value.SetValue("")

	pairs := e.Pairs()
	if len(pairs) != 0 {
		t.Errorf("expected 0 pairs for empty rows, got %d", len(pairs))
	}
}

func TestEditorView(t *testing.T) {
	e := NewKeyValueEditor("Header", "Value")
	e.SetWidth(60)
	e.Rows[0].key.SetValue("X-Test")
	e.Rows[0].value.SetValue("hello")

	view := e.View(10)
	if !strings.Contains(view, "X-Test") {
		t.Error("expected X-Test in view")
	}
	if !strings.Contains(view, "hello") {
		t.Error("expected hello in view")
	}
	if !strings.Contains(view, "Add Row") {
		t.Error("expected Add Row button in view")
	}
}

func TestEditorSetWidth(t *testing.T) {
	e := NewKeyValueEditor("K", "V")
	e.SetWidth(80)
	if e.Width != 80 {
		t.Errorf("expected Width=80, got %d", e.Width)
	}
}
