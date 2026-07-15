package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type KeyValuePair struct {
	Key   string
	Value string
}

type KeyValueEditor struct {
	Rows     []*kvRow
	FocusIdx int
	Width    int
	KeyLabel string
	ValLabel string
	focused  bool
}

type kvRow struct {
	key   textinput.Model
	value textinput.Model
}

func NewKeyValueEditor(keyLabel, valLabel string) *KeyValueEditor {
	e := &KeyValueEditor{
		KeyLabel: keyLabel,
		ValLabel: valLabel,
		FocusIdx: -1,
	}
	e.addEmptyRow()
	return e
}

func (e *KeyValueEditor) addEmptyRow() {
	k := textinput.New()
	k.Placeholder = e.KeyLabel
	k.CharLimit = 256
	v := textinput.New()
	v.Placeholder = e.ValLabel
	v.CharLimit = 4096
	e.Rows = append(e.Rows, &kvRow{key: k, value: v})
}

func (e *KeyValueEditor) removeRow(i int) {
	if i >= 0 && i < len(e.Rows) && len(e.Rows) > 1 {
		e.Rows = append(e.Rows[:i], e.Rows[i+1:]...)
	}
}

func (e *KeyValueEditor) SetWidth(w int) {
	e.Width = w
	keyW := w * 2 / 5
	if keyW < 15 {
		keyW = 15
	}
	valW := w - keyW - 6
	if valW < 20 {
		valW = 20
	}
	for _, row := range e.Rows {
		row.key.Width = keyW - 1
		row.value.Width = valW - 1
	}
}

func (e *KeyValueEditor) Focus() {
	e.focused = true
	if e.FocusIdx < 0 && len(e.Rows) > 0 {
		e.FocusIdx = 0
	}
	e.syncFocus()
}

func (e *KeyValueEditor) Blur() {
	e.focused = false
	e.FocusIdx = -1
	for _, row := range e.Rows {
		row.key.Blur()
		row.value.Blur()
	}
}

func (e *KeyValueEditor) Focused() bool {
	return e.focused
}

func (e *KeyValueEditor) syncFocus() {
	for i, row := range e.Rows {
		if e.FocusIdx == i*2 {
			row.key.Focus()
			row.value.Blur()
		} else if e.FocusIdx == i*2+1 {
			row.key.Blur()
			row.value.Focus()
		} else {
			row.key.Blur()
			row.value.Blur()
		}
	}
}

func (e *KeyValueEditor) totalFocusable() int {
	return len(e.Rows)*2 + 1
}

func (e *KeyValueEditor) HandleKey(msg tea.KeyMsg) {
	if !e.focused || e.FocusIdx < 0 {
		return
	}

	switch msg.String() {
	case "tab":
		e.FocusIdx = (e.FocusIdx + 1) % e.totalFocusable()
		e.syncFocus()
		return
	case "shift+tab":
		e.FocusIdx = (e.FocusIdx - 1 + e.totalFocusable()) % e.totalFocusable()
		e.syncFocus()
		return
	case "enter":
		if e.FocusIdx == len(e.Rows)*2 {
			e.addEmptyRow()
			return
		}
	case "backspace", "ctrl+backspace":
		if e.FocusIdx < len(e.Rows)*2 {
			rowIdx := e.FocusIdx / 2
			if e.Rows[rowIdx].key.Value() == "" {
				e.removeRow(rowIdx)
				if e.FocusIdx >= e.totalFocusable() {
					e.FocusIdx = e.totalFocusable() - 1
				}
				e.syncFocus()
				return
			}
		}
	}

	if e.FocusIdx < len(e.Rows)*2 {
		rowIdx := e.FocusIdx / 2
		fieldIdx := e.FocusIdx % 2
		if fieldIdx == 0 {
			e.Rows[rowIdx].key.Update(msg)
		} else {
			e.Rows[rowIdx].value.Update(msg)
		}
	}
}

func (e *KeyValueEditor) Pairs() []KeyValuePair {
	var pairs []KeyValuePair
	for _, row := range e.Rows {
		k := strings.TrimSpace(row.key.Value())
		v := strings.TrimSpace(row.value.Value())
		if k != "" || v != "" {
			pairs = append(pairs, KeyValuePair{Key: k, Value: v})
		}
	}
	return pairs
}

func (e *KeyValueEditor) ToMap() map[string]string {
	m := make(map[string]string)
	for _, p := range e.Pairs() {
		if p.Key != "" {
			m[p.Key] = p.Value
		}
	}
	return m
}

func (e *KeyValueEditor) RowCount() int {
	return len(e.Rows)
}

var (
	editorSep = lipgloss.NewStyle().Foreground(borderCol).Render(" │ ")
)

func (e *KeyValueEditor) View(height int) string {
	if e.Width == 0 {
		e.Width = 60
	}
	keyW := e.Width * 2 / 5
	if keyW < 14 {
		keyW = 14
	}
	valW := e.Width - keyW - 5
	if valW < 18 {
		valW = 18
	}

	var b strings.Builder

	hdrStyle := lipgloss.NewStyle().Bold(true).Foreground(secondary).Padding(0, 1)
	b.WriteString(hdrStyle.Render(fmt.Sprintf("%-*s │ %-*s", keyW, e.KeyLabel, valW, e.ValLabel)))
	b.WriteString("\n")

	for i, row := range e.Rows {
		if i >= height-3 && height > 3 {
			break
		}
		rowStyle := lipgloss.NewStyle().Padding(0, 1)
		if e.focused && e.FocusIdx/2 == i {
			rowStyle = lipgloss.NewStyle().Padding(0, 1).Background(lipgloss.Color("#0F3460"))
		}

		row.key.Width = keyW - 1
		row.value.Width = valW - 1

		keyView := row.key.View()
		valView := row.value.View()
		line := fmt.Sprintf("%-*s%s%-*s", keyW, keyView, editorSep, valW, valView)
		b.WriteString(rowStyle.Render(line))
		if i < len(e.Rows)-1 {
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	btnStyle := editorAddBtn
	if e.focused && e.FocusIdx == len(e.Rows)*2 {
		btnStyle = editorAddBtnFocus
	}
	b.WriteString(btnStyle.Render("[ + Add Row ]"))

	return b.String()
}
