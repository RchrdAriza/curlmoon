package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestBuildURL_NoParams(t *testing.T) {
	m := NewModel()
	m.urlInput.SetValue("https://httpbin.org/get")
	url := m.buildURL()
	if url != "https://httpbin.org/get" {
		t.Errorf("expected unchanged URL, got %s", url)
	}
}

func TestBuildURL_WithParams(t *testing.T) {
	m := NewModel()
	m.urlInput.SetValue("https://httpbin.org/get")
	m.params.Rows[0].key.SetValue("name")
	m.params.Rows[0].value.SetValue("test")

	url := m.buildURL()
	if !strings.Contains(url, "name=test") {
		t.Errorf("expected name=test in URL, got %s", url)
	}
	if !strings.Contains(url, "?") {
		t.Errorf("expected ? in URL with params, got %s", url)
	}
}

func TestBuildURL_WithExistingQuery(t *testing.T) {
	m := NewModel()
	m.urlInput.SetValue("https://httpbin.org/get?existing=1")
	m.params.Rows[0].key.SetValue("page")
	m.params.Rows[0].value.SetValue("2")

	url := m.buildURL()
	if !strings.Contains(url, "existing=1") {
		t.Errorf("expected existing param preserved, got %s", url)
	}
	if !strings.Contains(url, "page=2") {
		t.Errorf("expected page=2 in URL, got %s", url)
	}
	if !strings.Contains(url, "&") {
		t.Errorf("expected & when appending params, got %s", url)
	}
}

func TestBuildHeaders_NoBody(t *testing.T) {
	m := NewModel()
	m.bodyType = 0
	headers := m.buildHeaders()
	if len(headers) > 0 {
		t.Errorf("expected no headers for no body, got %v", headers)
	}
}

func TestBuildHeaders_JSONBody(t *testing.T) {
	m := NewModel()
	m.bodyType = 1
	headers := m.buildHeaders()
	if headers["Content-Type"] != "application/json" {
		t.Errorf("expected application/json, got %s", headers["Content-Type"])
	}
}

func TestBuildHeaders_RawBody(t *testing.T) {
	m := NewModel()
	m.bodyType = 2
	headers := m.buildHeaders()
	if headers["Content-Type"] != "text/plain" {
		t.Errorf("expected text/plain, got %s", headers["Content-Type"])
	}
}

func TestBuildHeaders_UserOverride(t *testing.T) {
	m := NewModel()
	m.bodyType = 1
	m.headers.Rows[0].key.SetValue("Content-Type")
	m.headers.Rows[0].value.SetValue("application/vnd.api+json")
	headers := m.buildHeaders()
	if headers["Content-Type"] != "application/vnd.api+json" {
		t.Errorf("expected user override, got %s", headers["Content-Type"])
	}
}

func TestBuildHeaders_CustomHeaders(t *testing.T) {
	m := NewModel()
	m.headers.Rows[0].key.SetValue("Authorization")
	m.headers.Rows[0].value.SetValue("Bearer mytoken")
	headers := m.buildHeaders()
	if headers["Authorization"] != "Bearer mytoken" {
		t.Errorf("expected Bearer mytoken, got %s", headers["Authorization"])
	}
}

func TestBuildBody_None(t *testing.T) {
	m := NewModel()
	m.bodyType = 0
	body := m.buildBody()
	if body != "" {
		t.Errorf("expected empty body for type none, got %s", body)
	}
}

func TestBuildBody_JSON(t *testing.T) {
	m := NewModel()
	m.bodyType = 1
	m.bodyEditor.SetValue(`{"hello":"world"}`)
	body := m.buildBody()
	if body != `{"hello":"world"}` {
		t.Errorf("expected JSON body, got %s", body)
	}
}

func TestModelSubFocus_EnterExit(t *testing.T) {
	m := NewModel()
	m = m.initLayout(100, 30)
	m.activeTab = 0

	if m.subFocus {
		t.Error("expected subFocus=false initially")
	}

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := result.(Model)

	if !m2.subFocus {
		t.Error("expected subFocus=true after Enter on headers tab")
	}
	if !m2.headers.Focused() {
		t.Error("expected headers editor focused")
	}

	result, _ = m2.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m3 := result.(Model)

	if m3.subFocus {
		t.Error("expected subFocus=false after Esc")
	}
	if m3.headers.Focused() {
		t.Error("expected headers blurred after exit")
	}
}

func TestModelBodyTypeChange(t *testing.T) {
	m := NewModel()
	if m.bodyType != 0 {
		t.Errorf("expected bodyType=0 (none), got %d", m.bodyType)
	}
}

func TestModelSendWithHeaders(t *testing.T) {
	m := NewModel()
	m = m.initLayout(100, 30)
	m.urlInput.SetValue("https://httpbin.org/post")
	m.methodIndex = 1
	m.headers.Rows[0].key.SetValue("X-Custom")
	m.headers.Rows[0].value.SetValue("test123")
	m.bodyType = 1
	m.bodyEditor.SetValue(`{"data":"test"}`)

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	m2 := result.(Model)

	if !m2.sending {
		t.Error("expected sending=true")
	}

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
		t.Fatal("expected response")
	}
	if m3.response.StatusCode != 200 {
		t.Errorf("expected 200, got %d", m3.response.StatusCode)
	}
}

func TestModelParamsUpdateURL(t *testing.T) {
	m := NewModel()
	m.urlInput.SetValue("https://httpbin.org/get")
	m.params.Rows[0].key.SetValue("q")
	m.params.Rows[0].value.SetValue("search")

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	m2 := result.(Model)

	msg := cmd()
	respMsg := msg.(responseMsg)
	result, _ = m2.Update(respMsg)
	m3 := result.(Model)

	if m3.response == nil {
		t.Fatal("expected response")
	}
	if !strings.Contains(m3.response.Body, "search") {
		t.Errorf("expected 'search' in response args, got:\n%s", m3.response.Body)
	}
}

func TestModelTab_InsideEditor(t *testing.T) {
	m := NewModel()
	m = m.initLayout(100, 30)
	m.activeTab = 0
	m.subFocus = true
	m.headers.Focus()
	m.headers.FocusIdx = 0

	// Tab should be consumed by editor, not switch panels
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m2 := result.(Model)

	if m2.activePanel != panelRequest {
		t.Errorf("expected to stay on request panel, got %d", m2.activePanel)
	}
}

func TestModelTab_OutsideEditor(t *testing.T) {
	m := NewModel()
	m = m.initLayout(100, 30)
	m.activePanel = panelRequest
	m.subFocus = false

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m2 := result.(Model)

	if m2.activePanel == panelRequest {
		t.Error("expected panel to change when Tab outside editor")
	}
}
