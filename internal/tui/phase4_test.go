package tui

import (
	"curlmoon/internal/collection"
	"curlmoon/internal/environment"
	"strings"
	"testing"
)

// --- Auth helpers ---

func TestBuildHeaders_AuthNone(t *testing.T) {
	a := NewApp()
	a.authType = authNone
	headers := a.buildHeaders()
	if _, ok := headers["Authorization"]; ok {
		t.Errorf("expected no Authorization header, got %v", headers)
	}
}

func TestBuildHeaders_AuthBasic(t *testing.T) {
	a := NewApp()
	a.authType = authBasic
	a.authText = "Username: alice\nPassword: secret"
	headers := a.buildHeaders()
	if !strings.HasPrefix(headers["Authorization"], "Basic ") {
		t.Fatalf("expected Basic auth header, got %v", headers["Authorization"])
	}
}

func TestBuildHeaders_AuthBearer(t *testing.T) {
	a := NewApp()
	a.authType = authBearer
	a.authText = "abc123"
	headers := a.buildHeaders()
	if headers["Authorization"] != "Bearer abc123" {
		t.Errorf("expected Bearer abc123, got %q", headers["Authorization"])
	}
}

func TestBuildHeaders_AuthAPIKey(t *testing.T) {
	a := NewApp()
	a.authType = authAPIKey
	a.authText = "Key: X-API-Key\nValue: mykey"
	headers := a.buildHeaders()
	if headers["X-API-Key"] != "mykey" {
		t.Errorf("expected X-API-Key=mykey, got %v", headers)
	}
}

func TestBuildHeaders_AuthOAuth2(t *testing.T) {
	a := NewApp()
	a.authType = authOAuth2
	a.authText = "tok"
	headers := a.buildHeaders()
	if headers["Authorization"] != "Bearer tok" {
		t.Errorf("expected Bearer tok, got %q", headers["Authorization"])
	}
}

func TestEnterContentEditor_AuthTabNowEditable(t *testing.T) {
	a := NewApp()
	a.activeTab = tabAuth
	if !a.EnterContentEditor() {
		t.Fatal("expected Auth tab to be editable")
	}
}

// --- Environments ---

func newEnvApp(t *testing.T) (*App, *collection.Store) {
	t.Helper()
	store := collection.NewStore(t.TempDir())
	a := NewAppWithStore(store)
	return a, store
}

func TestConfirmPrompt_NewEnvironment(t *testing.T) {
	a, _ := newEnvApp(t)
	a.StartPrompt("newEnvironment", sidebarEntry{}, "local")
	a.ConfirmPrompt()

	if len(a.environments) != 1 || a.environments[0].Name != "local" {
		t.Fatalf("expected environment 'local' to be created, got %+v", a.environments)
	}
}

func TestToggleActiveEnvironment(t *testing.T) {
	a, _ := newEnvApp(t)
	a.StartPrompt("newEnvironment", sidebarEntry{}, "local")
	a.ConfirmPrompt()

	a.toggleActiveEnvironment(0)
	if a.activeEnvName != "local" {
		t.Fatalf("expected active environment 'local', got %q", a.activeEnvName)
	}
	a.toggleActiveEnvironment(0)
	if a.activeEnvName != "" {
		t.Fatalf("expected environment deactivated, got %q", a.activeEnvName)
	}
}

func TestStartAndSaveEnvEdit(t *testing.T) {
	a, _ := newEnvApp(t)
	a.StartPrompt("newEnvironment", sidebarEntry{}, "local")
	a.ConfirmPrompt()

	if !a.StartEnvEdit(0) {
		t.Fatal("expected StartEnvEdit to succeed")
	}
	a.envEditText = "host: localhost\nport: 8080"
	a.SaveEnvEdit()

	if a.envEditIdx != -1 {
		t.Errorf("expected envEditIdx reset to -1, got %d", a.envEditIdx)
	}
	vars := a.environments[0].Vars()
	if vars["host"] != "localhost" || vars["port"] != "8080" {
		t.Errorf("expected saved vars, got %v", vars)
	}
}

func TestDoRequest_ResolvesVariablesFromActiveEnvironment(t *testing.T) {
	a, _ := newEnvApp(t)
	a.StartPrompt("newEnvironment", sidebarEntry{}, "local")
	a.ConfirmPrompt()
	a.StartEnvEdit(0)
	a.envEditText = "base: https://example.com"
	a.SaveEnvEdit()
	a.toggleActiveEnvironment(0)

	a.urlValue = "{{base}}/get"
	vars := a.activeEnvVars()
	resolved := environment.Resolve(a.buildURL(), vars)
	if resolved != "https://example.com/get" {
		t.Errorf("expected resolved URL, got %q", resolved)
	}
}

func TestSelectSidebarEntry_EnvSection(t *testing.T) {
	a, _ := newEnvApp(t)
	a.StartPrompt("newEnvironment", sidebarEntry{}, "local")
	a.ConfirmPrompt()

	// Find the env entry in the freshly rebuilt sidebar.
	idx := -1
	for i, e := range a.sidebar {
		if e.section == "env" && !e.isFolder {
			idx = i
			break
		}
	}
	if idx == -1 {
		t.Fatal("expected an environment entry in the sidebar")
	}
	a.sidebarSel = idx
	if a.SelectSidebarEntry() {
		t.Error("expected selecting an environment entry to not load a request")
	}
	if a.activeEnvName != "local" {
		t.Errorf("expected environment activated, got %q", a.activeEnvName)
	}
}

// --- History ---

func TestHandleResponse_RecordsHistory(t *testing.T) {
	a, _ := newEnvApp(t)
	a.urlValue = "https://example.com"
	a.methodIndex = 0

	a.HandleResponse(nil, errQuoted("boom"))
	if len(a.historyEntries) != 1 {
		t.Fatalf("expected 1 history entry after error, got %d", len(a.historyEntries))
	}
	if a.historyEntries[0].URL != "https://example.com" {
		t.Errorf("expected recorded URL, got %q", a.historyEntries[0].URL)
	}
}

func TestLoadHistoryEntry(t *testing.T) {
	a, _ := newEnvApp(t)
	a.urlValue = "https://example.com/foo"
	a.methodIndex = 1 // POST
	a.HandleResponse(nil, errQuoted("boom"))

	a.urlValue = ""
	a.methodIndex = 0
	if !a.loadHistoryEntry(0) {
		t.Fatal("expected loadHistoryEntry to succeed")
	}
	if a.urlValue != "https://example.com/foo" || methods[a.methodIndex] != "POST" {
		t.Errorf("expected restored request, got url=%q method=%q", a.urlValue, methods[a.methodIndex])
	}
}
