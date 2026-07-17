package tui

import (
	"strings"
	"testing"
)

func TestNewApp(t *testing.T) {
	a := NewApp()
	if a.activePanel != panelURL {
		t.Errorf("expected activePanel=%s, got %s", panelURL, a.activePanel)
	}
	if a.methodIndex != 0 {
		t.Errorf("expected methodIndex=0 (GET), got %d", a.methodIndex)
	}
	if len(a.sidebar) == 0 {
		t.Error("expected sidebar to have items")
	}
}

func TestAppCycleMethod(t *testing.T) {
	a := NewApp()
	a.CycleMethod(1)
	if a.methodIndex != 1 {
		t.Errorf("expected methodIndex=1 (POST), got %d", a.methodIndex)
	}
	a.CycleMethod(-1)
	if a.methodIndex != 0 {
		t.Errorf("expected methodIndex=0 (GET), got %d", a.methodIndex)
	}
	a.CycleMethod(-1)
	if a.methodIndex != len(methods)-1 {
		t.Errorf("expected wrap to last method, got %d", a.methodIndex)
	}
}

func TestAppTabNavigation(t *testing.T) {
	a := NewApp()
	a.NextTab()
	if a.activeTab != tabBody {
		t.Errorf("expected activeTab=%d, got %d", tabBody, a.activeTab)
	}
	a.PrevTab()
	if a.activeTab != tabHeaders {
		t.Errorf("expected activeTab=%d, got %d", tabHeaders, a.activeTab)
	}
	// Doesn't wrap past the ends.
	a.PrevTab()
	if a.activeTab != tabHeaders {
		t.Errorf("expected activeTab to stay at %d, got %d", tabHeaders, a.activeTab)
	}
}

func TestAppSidebarNavigation(t *testing.T) {
	a := NewApp()
	a.MoveSidebarSel(1, 100)
	if a.sidebarSel != 1 {
		t.Errorf("expected sidebarSel=1 after down, got %d", a.sidebarSel)
	}
	a.MoveSidebarSel(-1, 100)
	if a.sidebarSel != 0 {
		t.Errorf("expected sidebarSel=0 after up, got %d", a.sidebarSel)
	}
	a.MoveSidebarSel(-1, 100)
	if a.sidebarSel != 0 {
		t.Error("expected sidebarSel to stay at 0 when already at top")
	}
}

func TestAppSidebarSelectItem(t *testing.T) {
	a := NewApp()
	a.sidebarSel = 1 // "GET /get"
	if !a.SelectSidebarEntry() {
		t.Fatal("expected a request entry to be selected")
	}
	if a.activePanel != panelURL {
		t.Errorf("expected focus to switch to url panel, got %s", a.activePanel)
	}
	if a.urlValue != "https://httpbin.org/get" {
		t.Errorf("expected URL to be httpbin.org/get, got %s", a.urlValue)
	}
}

func TestAppSidebarSelectFolder(t *testing.T) {
	a := NewApp()
	a.sidebarSel = 0 // "httpbin.org" folder
	if a.SelectSidebarEntry() {
		t.Error("expected selecting a folder to be a no-op")
	}
}

func TestAppEnterExitContentEditor(t *testing.T) {
	a := NewApp()
	a.activeTab = tabHeaders
	if a.subFocus {
		t.Error("expected subFocus=false initially")
	}
	if !a.EnterContentEditor() {
		t.Fatal("expected EnterContentEditor to succeed on Headers tab")
	}
	if !a.subFocus {
		t.Error("expected subFocus=true after EnterContentEditor")
	}
	a.ExitContentEditor()
	if a.subFocus {
		t.Error("expected subFocus=false after ExitContentEditor")
	}
}

func TestAppEnterContentEditor_AuthTabBlocked(t *testing.T) {
	a := NewApp()
	a.activeTab = tabAuth
	if a.EnterContentEditor() {
		t.Error("expected EnterContentEditor to be blocked on Auth tab")
	}
	if a.subFocus {
		t.Error("expected subFocus to remain false")
	}
}

func TestBuildURL_NoParams(t *testing.T) {
	a := NewApp()
	a.urlValue = "https://httpbin.org/get"
	if got := a.buildURL(); got != "https://httpbin.org/get" {
		t.Errorf("expected unchanged URL, got %s", got)
	}
}

func TestBuildURL_WithParams(t *testing.T) {
	a := NewApp()
	a.urlValue = "https://httpbin.org/get"
	a.paramsText = "name: test"

	got := a.buildURL()
	if !strings.Contains(got, "name=test") {
		t.Errorf("expected name=test in URL, got %s", got)
	}
	if !strings.Contains(got, "?") {
		t.Errorf("expected ? in URL with params, got %s", got)
	}
}

func TestBuildURL_WithExistingQuery(t *testing.T) {
	a := NewApp()
	a.urlValue = "https://httpbin.org/get?existing=1"
	a.paramsText = "page: 2"

	got := a.buildURL()
	if !strings.Contains(got, "existing=1") {
		t.Errorf("expected existing param preserved, got %s", got)
	}
	if !strings.Contains(got, "page=2") {
		t.Errorf("expected page=2 in URL, got %s", got)
	}
	if !strings.Contains(got, "&") {
		t.Errorf("expected & when appending params, got %s", got)
	}
}

func TestBuildHeaders_NoBody(t *testing.T) {
	a := NewApp()
	a.bodyType = 0
	if headers := a.buildHeaders(); len(headers) > 0 {
		t.Errorf("expected no headers for no body, got %v", headers)
	}
}

func TestBuildHeaders_JSONBody(t *testing.T) {
	a := NewApp()
	a.bodyType = 1
	headers := a.buildHeaders()
	if headers["Content-Type"] != "application/json" {
		t.Errorf("expected application/json, got %s", headers["Content-Type"])
	}
}

func TestBuildHeaders_RawBody(t *testing.T) {
	a := NewApp()
	a.bodyType = 2
	headers := a.buildHeaders()
	if headers["Content-Type"] != "text/plain" {
		t.Errorf("expected text/plain, got %s", headers["Content-Type"])
	}
}

func TestBuildHeaders_UserOverride(t *testing.T) {
	a := NewApp()
	a.bodyType = 1
	a.headersText = "Content-Type: application/vnd.api+json"
	headers := a.buildHeaders()
	if headers["Content-Type"] != "application/vnd.api+json" {
		t.Errorf("expected user override, got %s", headers["Content-Type"])
	}
}

func TestBuildHeaders_CustomHeaders(t *testing.T) {
	a := NewApp()
	a.headersText = "Authorization: Bearer mytoken"
	headers := a.buildHeaders()
	if headers["Authorization"] != "Bearer mytoken" {
		t.Errorf("expected Bearer mytoken, got %s", headers["Authorization"])
	}
}

func TestBuildBody_None(t *testing.T) {
	a := NewApp()
	a.bodyType = 0
	if body := a.buildBody(); body != "" {
		t.Errorf("expected empty body for type none, got %s", body)
	}
}

func TestBuildBody_JSON(t *testing.T) {
	a := NewApp()
	a.bodyType = 1
	a.bodyText = `{"hello":"world"}`
	if body := a.buildBody(); body != `{"hello":"world"}` {
		t.Errorf("expected JSON body, got %s", body)
	}
}

func TestAppQuitSavesSession(t *testing.T) {
	// saveSession is a no-op without a store; just verify it doesn't panic.
	a := NewApp()
	a.urlValue = "https://example.com"
	a.saveSession()
}

func TestAppHandleResponse_Error(t *testing.T) {
	a := NewApp()
	a.HandleResponse(nil, errQuoted("boom"))
	if a.respErr == nil {
		t.Error("expected respErr to be set")
	}
	if a.showResp {
		t.Error("expected showResp=false on error")
	}
	if !strings.Contains(a.statusMsg, "Error") {
		t.Error("expected statusMsg to mention the error")
	}
}

type errQuoted string

func (e errQuoted) Error() string { return string(e) }
