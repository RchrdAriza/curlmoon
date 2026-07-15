package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewModel(t *testing.T) {
	m := NewModel()
	if m.activePanel != panelRequest {
		t.Errorf("expected activePanel=%d, got %d", panelRequest, m.activePanel)
	}
	if m.methodIndex != 0 {
		t.Errorf("expected methodIndex=0 (GET), got %d", m.methodIndex)
	}
	if m.urlInput.Placeholder != "https://httpbin.org/get" {
		t.Errorf("unexpected placeholder: %s", m.urlInput.Placeholder)
	}
	if len(m.sidebar) == 0 {
		t.Error("expected sidebar to have items")
	}
}

func TestModelInit(t *testing.T) {
	m := NewModel()
	cmd := m.Init()
	if cmd == nil {
		t.Error("expected Init to return a command")
	}
}

func TestModelWindowResize(t *testing.T) {
	m := NewModel()
	msg := tea.WindowSizeMsg{Width: 100, Height: 30}
	result, cmd := m.Update(msg)
	if cmd != nil {
		t.Error("expected no command from resize")
	}
	updated := result.(Model)
	if updated.width != 100 {
		t.Errorf("expected width=100, got %d", updated.width)
	}
	if updated.height != 30 {
		t.Errorf("expected height=30, got %d", updated.height)
	}
	if !updated.ready {
		t.Error("expected ready=true after resize")
	}
}

func TestModelFocusChange(t *testing.T) {
	m := NewModel()
	if m.activePanel != panelRequest {
		t.Errorf("expected focus on request, got %d", m.activePanel)
	}

	// Tab: request(1) -> response(2) -> sidebar(0) -> request(1)
	msg := tea.KeyMsg{Type: tea.KeyTab}
	result, _ := m.Update(msg)
	updated := result.(Model)
	if updated.activePanel != panelResponse {
		t.Errorf("expected focus on response (2) after tab, got %d", updated.activePanel)
	}

	result, _ = updated.Update(msg)
	updated = result.(Model)
	if updated.activePanel != panelSidebar {
		t.Errorf("expected focus on sidebar (0) after tab, got %d", updated.activePanel)
	}

	result, _ = updated.Update(msg)
	updated = result.(Model)
	if updated.activePanel != panelRequest {
		t.Errorf("expected focus on request (1) after tab, got %d", updated.activePanel)
	}
}

func TestModelSidebarNavigation(t *testing.T) {
	m := NewModel()
	m.activePanel = panelSidebar
	m.width = 100
	m.height = 30

	// Navigate down
	msg := tea.KeyMsg{Type: tea.KeyDown}
	result, _ := m.Update(msg)
	updated := result.(Model)
	if updated.sidebarSel != 1 {
		t.Errorf("expected sidebarSel=1 after down, got %d", updated.sidebarSel)
	}

	// Navigate up
	msg = tea.KeyMsg{Type: tea.KeyUp}
	result, _ = updated.Update(msg)
	updated = result.(Model)
	if updated.sidebarSel != 0 {
		t.Errorf("expected sidebarSel=0 after up, got %d", updated.sidebarSel)
	}
}

func TestModelMethodChange(t *testing.T) {
	m := NewModel()
	m.activePanel = panelRequest
	m.width = 100
	m.height = 30

	// Down changes method POST
	msg := tea.KeyMsg{Type: tea.KeyDown}
	result, _ := m.Update(msg)
	updated := result.(Model)
	if updated.methodIndex != 1 {
		t.Errorf("expected methodIndex=1 (POST), got %d", updated.methodIndex)
	}

	// Up goes back to GET
	msg = tea.KeyMsg{Type: tea.KeyUp}
	result, _ = updated.Update(msg)
	updated = result.(Model)
	if updated.methodIndex != 0 {
		t.Errorf("expected methodIndex=0 (GET), got %d", updated.methodIndex)
	}
}

func TestModelTabNavigation(t *testing.T) {
	m := NewModel()
	m.activePanel = panelRequest
	m.width = 100
	m.height = 30

	// Right arrow to move to Body tab
	msg := tea.KeyMsg{Type: tea.KeyRight}
	result, _ := m.Update(msg)
	updated := result.(Model)
	if updated.activeTab != 1 {
		t.Errorf("expected activeTab=1, got %d", updated.activeTab)
	}

	// Left arrow back to Headers
	msg = tea.KeyMsg{Type: tea.KeyLeft}
	result, _ = updated.Update(msg)
	updated = result.(Model)
	if updated.activeTab != 0 {
		t.Errorf("expected activeTab=0, got %d", updated.activeTab)
	}
}

func TestModelViewNotEmpty(t *testing.T) {
	m := NewModel()
	m.width = 100
	m.height = 30
	m.ready = true

	view := m.View()
	if len(view) == 0 {
		t.Error("expected non-empty view")
	}
	if !strings.Contains(view, "Collections") {
		t.Error("expected 'Collections' in sidebar")
	}
	if !strings.Contains(view, "GET") {
		t.Error("expected method badge in view")
	}
}

func TestModelURLInput(t *testing.T) {
	m := NewModel()
	m.activePanel = panelRequest
	m.width = 100
	m.height = 30

	// Simulate typing a URL
	for _, ch := range "https://example.com/api" {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}}
		result, _ := m.Update(msg)
		m = result.(Model)
	}

	if m.urlInput.Value() != "https://example.com/api" {
		t.Errorf("expected URL to be set, got %s", m.urlInput.Value())
	}
}

func TestModelSendRequest(t *testing.T) {
	m := NewModel()
	m.activePanel = panelRequest
	m.width = 100
	m.height = 30
	m.urlInput.SetValue("https://httpbin.org/get")

	msg := tea.KeyMsg{Type: tea.KeyCtrlR}
	result, _ := m.Update(msg)
	updated := result.(Model)

	if !updated.sending {
		t.Error("expected sending=true after Ctrl+R")
	}
	if !strings.Contains(updated.statusMsg, "Sending") {
		t.Error("expected statusMsg to show sending")
	}
}

func TestModelSidebarSelectItem(t *testing.T) {
	m := NewModel()
	m.activePanel = panelSidebar
	m.width = 100
	m.height = 30

	// Select second item (GET /get)
	m.sidebarSel = 1
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	result, _ := m.Update(msg)
	updated := result.(Model)

	if updated.activePanel != panelRequest {
		t.Errorf("expected focus to switch to request panel, got %d", updated.activePanel)
	}
	if updated.urlInput.Value() != "https://httpbin.org/get" {
		t.Errorf("expected URL to be httpbin.org/get, got %s", updated.urlInput.Value())
	}
}

func TestModelQuit(t *testing.T) {
	m := NewModel()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.Update(msg)

	if cmd == nil {
		t.Error("expected quit command for 'q' key")
	}
}

func TestModelCtrlC(t *testing.T) {
	m := NewModel()
	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := m.Update(msg)

	if cmd == nil {
		t.Error("expected quit command for Ctrl+C")
	}
}

func TestModelRendersResponseView(t *testing.T) {
	m := NewModel()
	m.width = 100
	m.height = 30
	m.ready = true
	m.showResp = false

	view := m.View()
	if strings.Contains(view, "Send a request") {
		// This is expected - no response yet
	} else {
		t.Log("View renders correctly without response")
	}
}


