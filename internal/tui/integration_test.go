package tui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestIntegration_FullRequestResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","data":[1,2,3]}`))
	}))
	defer srv.Close()

	m := NewModel()
	m = m.initLayout(120, 40)
	m.urlInput.SetValue(srv.URL)
	m.methodIndex = 0

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	if cmd == nil {
		t.Fatal("expected command from send")
	}
	m2 := result.(Model)

	msg := cmd()
	respMsg, ok := msg.(responseMsg)
	if !ok {
		t.Fatalf("expected responseMsg, got %T", msg)
	}
	if respMsg.err != nil {
		t.Fatalf("request failed: %v", respMsg.err)
	}

	result, _ = m2.Update(respMsg)
	m3 := result.(Model)

	if m3.response == nil {
		t.Fatal("expected response to be set")
	}
	if m3.response.StatusCode != 200 {
		t.Errorf("expected 200, got %d", m3.response.StatusCode)
	}
	if !strings.Contains(m3.response.Body, "status") {
		t.Errorf("expected response body, got: %s", m3.response.Body)
	}
	if len(m3.response.Headers) == 0 {
		t.Error("expected response headers")
	}
	if m3.response.Size <= 0 {
		t.Error("expected positive size")
	}
	if m3.response.Elapsed <= 0 {
		t.Error("expected positive elapsed time")
	}
}

func TestIntegration_ErrorHandling(t *testing.T) {
	m := NewModel()
	m.width = 120
	m.height = 40
	m.ready = true

	// Set invalid URL
	m.urlInput.SetValue("http://nonexistent.invalid/api")
	m.methodIndex = 0

	// Send
	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	m2 := result.(Model)

	// Execute
	msg := cmd()
	respMsg := msg.(responseMsg)

	// Process error
	result, _ = m2.Update(respMsg)
	m3 := result.(Model)

	if m3.respErr == nil {
		t.Error("expected error for invalid URL")
	}
	if m3.showResp {
		t.Error("expected showResp=false on error")
	}
	if !strings.Contains(m3.statusMsg, "Error") {
		t.Error("expected statusMsg to show error")
	}
}

func TestIntegration_SuccessResponseView(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result":"success"}`))
	}))
	defer srv.Close()

	m := NewModel()
	m = m.initLayout(120, 40)
	m.urlInput.SetValue(srv.URL)

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	m2 := result.(Model)
	msg := cmd()
	respMsg := msg.(responseMsg)
	result, _ = m2.Update(respMsg)
	m3 := result.(Model)

	view := m3.View()

	if !strings.Contains(view, "200") {
		t.Error("expected status code 200 in view")
	}
	if !strings.Contains(view, "success") {
		t.Error("expected response body in view")
	}
}
