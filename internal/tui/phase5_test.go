package tui

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"curlmoon/internal/codegen"
	"curlmoon/internal/collection"
	"curlmoon/internal/config"
)

// --- trimTrailingNewline (gocui trailing-null-cell workaround) ---

func TestTrimTrailingNewline_StripsPerLineTrailingSpace(t *testing.T) {
	// Simulates gocui's Buffer(): every line typed into ends up with one
	// trailing space (a null cell rendered as space), plus a trailing "\n"
	// after the whole buffer.
	got := trimTrailingNewline("query { \nme { id } \n} \n")
	want := "query {\nme { id }\n}"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTrimTrailingNewline_SingleLine(t *testing.T) {
	if got := trimTrailingNewline("https://httpbin.org/post \n"); got != "https://httpbin.org/post" {
		t.Errorf("got %q", got)
	}
}

func TestTrimTrailingNewline_Empty(t *testing.T) {
	if got := trimTrailingNewline(""); got != "" {
		t.Errorf("got %q", got)
	}
}

// --- GraphQL ---

func TestParseGraphQLBody_WithVariables(t *testing.T) {
	text := "query { me { id } }\n\n### variables\n{\"id\": 1}"
	query, vars := parseGraphQLBody(text)
	if query != "query { me { id } }" {
		t.Errorf("unexpected query: %q", query)
	}
	if vars != `{"id": 1}` {
		t.Errorf("unexpected variables: %q", vars)
	}
}

func TestParseGraphQLBody_NoVariables(t *testing.T) {
	query, vars := parseGraphQLBody("query { me { id } }")
	if query != "query { me { id } }" || vars != "" {
		t.Errorf("expected no variables, got query=%q vars=%q", query, vars)
	}
}

func TestBuildGraphQLBody(t *testing.T) {
	body, err := buildGraphQLBody("query { me { id } }\n### variables\n{\"id\": 1}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(body, `"query":"query { me { id } }"`) || !strings.Contains(body, `"variables":{"id":1}`) {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestBuildGraphQLBody_InvalidVariablesJSON(t *testing.T) {
	if _, err := buildGraphQLBody("query { me }\n### variables\nnot json"); err == nil {
		t.Error("expected error for invalid variables JSON")
	}
}

func TestApp_BuildBody_GraphQL(t *testing.T) {
	a := NewApp()
	a.bodyType = bodyTypeGraphQL
	a.bodyText = "query { me { id } }"
	body := a.buildBody()
	if !strings.Contains(body, `"query":"query { me { id } }"`) {
		t.Errorf("expected query in body, got %s", body)
	}
}

func TestApp_BuildHeaders_GraphQLSetsJSONContentType(t *testing.T) {
	a := NewApp()
	a.bodyType = bodyTypeGraphQL
	headers := a.buildHeaders()
	if headers["Content-Type"] != "application/json" {
		t.Errorf("expected application/json, got %q", headers["Content-Type"])
	}
}

// --- Scripts ---

func TestParseScripts(t *testing.T) {
	text := "### pre-request\npm.environment.set(\"x\", \"1\");\n\n### test\npm.test(\"ok\", () => true);"
	pre, test := parseScripts(text)
	if pre != `pm.environment.set("x", "1");` {
		t.Errorf("unexpected pre-request: %q", pre)
	}
	if test != `pm.test("ok", () => true);` {
		t.Errorf("unexpected test: %q", test)
	}
}

func TestParseScripts_Empty(t *testing.T) {
	pre, test := parseScripts(defaultScriptsText)
	if pre != "" || test != "" {
		t.Errorf("expected empty sections in default text, got pre=%q test=%q", pre, test)
	}
}

func TestDoRequest_RunsTestScript(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	a := NewApp()
	a.urlValue = srv.URL
	a.scriptsText = "### test\npm.test(\"status is 200\", () => pm.response.code === 200);\npm.test(\"fails\", () => false);"

	resp, err := a.doRequest()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	a.HandleResponse(resp, err)

	if len(a.testResults) != 2 {
		t.Fatalf("expected 2 test results, got %d", len(a.testResults))
	}
	if !a.testResults[0].Passed {
		t.Errorf("expected first test to pass, got %+v", a.testResults[0])
	}
	if a.testResults[1].Passed {
		t.Errorf("expected second test to fail, got %+v", a.testResults[1])
	}
}

func TestDoRequest_PreRequestScriptSetsEnvVar(t *testing.T) {
	var gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Token")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a := NewApp()
	a.urlValue = srv.URL
	a.headersText = "X-Token: {{token}}"
	a.scriptsText = `### pre-request` + "\n" + `pm.environment.set("token", "abc123");`

	resp, err := a.doRequest()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	a.HandleResponse(resp, err)

	if gotHeader != "abc123" {
		t.Errorf("expected X-Token=abc123, got %q", gotHeader)
	}
}

func TestDoRequest_PreRequestScriptError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a := NewApp()
	a.urlValue = srv.URL
	a.scriptsText = "### pre-request\nthis is not valid js(((\n"

	resp, err := a.doRequest()
	a.HandleResponse(resp, err)

	if a.scriptErr == "" {
		t.Error("expected scriptErr to be set")
	}
}

func TestApp_CycleBodyType(t *testing.T) {
	a := NewApp()
	if a.bodyType != 0 {
		t.Fatalf("expected default bodyType=0, got %d", a.bodyType)
	}
	for i := 1; i < len(bodyTypes); i++ {
		a.CycleBodyType(1)
		if a.bodyType != i {
			t.Fatalf("expected bodyType=%d, got %d", i, a.bodyType)
		}
	}
	a.CycleBodyType(1)
	if a.bodyType != 0 {
		t.Errorf("expected wraparound to 0, got %d", a.bodyType)
	}
	if bodyTypes[bodyTypeGraphQL] != "GraphQL" {
		t.Errorf("expected GraphQL reachable via CycleBodyType, got bodyTypes=%v", bodyTypes)
	}
}

// --- Code generation overlay ---

func TestApp_ToggleCodegen(t *testing.T) {
	a := NewApp()
	if a.showCodegen {
		t.Fatal("expected codegen closed by default")
	}
	a.ToggleCodegen()
	if !a.showCodegen {
		t.Error("expected codegen open after toggle")
	}
	a.ToggleCodegen()
	if a.showCodegen {
		t.Error("expected codegen closed after second toggle")
	}
}

func TestApp_CodegenLangCycling(t *testing.T) {
	a := NewApp()
	if a.codegenLang != 0 {
		t.Fatalf("expected codegenLang=0 initially, got %d", a.codegenLang)
	}
	a.PrevCodegenLang()
	if a.codegenLang != len(codegen.Langs)-1 {
		t.Errorf("expected wraparound to last lang, got %d", a.codegenLang)
	}
	a.NextCodegenLang()
	if a.codegenLang != 0 {
		t.Errorf("expected wraparound back to 0, got %d", a.codegenLang)
	}
}

func TestApp_CodegenSnippet(t *testing.T) {
	a := NewApp()
	a.urlValue = "https://example.com/api"
	a.methodIndex = 0
	snippet := a.codegenSnippet()
	if !strings.Contains(snippet, "https://example.com/api") {
		t.Errorf("expected snippet to contain URL, got %s", snippet)
	}
}

// --- Theme ---

func TestApp_ToggleTheme(t *testing.T) {
	currentTheme = darkTheme
	a := NewApp()
	a.ToggleTheme()
	if themeName(currentTheme) != "light" {
		t.Errorf("expected light theme after toggle, got %s", themeName(currentTheme))
	}
	a.ToggleTheme()
	if themeName(currentTheme) != "dark" {
		t.Errorf("expected dark theme after second toggle, got %s", themeName(currentTheme))
	}
}

// --- Export / Import ---

func TestConfirmPrompt_ExportPath(t *testing.T) {
	dir := t.TempDir()
	store := collection.NewStore(dir)
	store.Create("Exportable")
	a := NewAppWithStore(store)

	sel := a.sidebar[sidebarIndexOf(a, "Exportable")]
	destPath := filepath.Join(dir, "out.json")
	a.StartPrompt("exportPath", sel, destPath)
	a.promptText = destPath
	a.ConfirmPrompt()

	if !strings.Contains(a.statusMsg, "Exported") {
		t.Errorf("expected export success message, got %q", a.statusMsg)
	}
	if _, err := os.Stat(destPath); err != nil {
		t.Errorf("expected exported file to exist: %v", err)
	}
}

func TestConfirmPrompt_ImportPath(t *testing.T) {
	dir := t.TempDir()
	importFile := filepath.Join(dir, "import.json")
	if err := os.WriteFile(importFile, []byte(`{"info":{"name":"Imported"},"item":[]}`), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	store := collection.NewStore(t.TempDir())
	a := NewAppWithStore(store)
	a.StartPrompt("importPath", sidebarEntry{}, importFile)
	a.promptText = importFile
	a.ConfirmPrompt()

	if !strings.Contains(a.statusMsg, "Imported") {
		t.Errorf("expected import success message, got %q", a.statusMsg)
	}
	found := false
	for _, c := range a.collections {
		if c.Info.Name == "Imported" {
			found = true
		}
	}
	if !found {
		t.Error("expected imported collection to appear in a.collections")
	}
}

// --- Keymap wiring ---

func TestNewAppWithStore_LoadsKeymapAndTheme(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "keybindings.json"), []byte(`{"quit": {"key": "ctrl+x"}}`), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"theme": "light"}`), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	store := collection.NewStore(dir)
	a := NewAppWithStore(store)
	if a.keymap["quit"].Key != "ctrl+x" {
		t.Errorf("expected overridden quit key, got %+v", a.keymap["quit"])
	}
	if themeName(currentTheme) != "light" {
		t.Errorf("expected light theme loaded, got %s", themeName(currentTheme))
	}
	currentTheme = darkTheme // reset for other tests
}

func TestDefaultKeymap_CoversEveryAction(t *testing.T) {
	// Sanity check that every action name keybindings.go's bind() calls
	// resolves to a real default, so a fresh install never falls through
	// to a zero-value key.
	actions := []string{
		"quit", "cycleFocus", "jumpSidebar", "jumpURL", "jumpResponse", "jumpContent", "sendRequest",
		"sidebarUp", "sidebarDown", "sidebarEnter", "sidebarNewCollection", "sidebarNewRequest",
		"sidebarRename", "sidebarDelete", "sidebarEditVars", "sidebarExport", "sidebarImport",
		"urlMethodUp", "urlMethodDown", "urlSwitchTabPrev", "urlSwitchTabNext", "urlHome", "urlEnd",
		"urlEnterContent", "cycleBodyType", "contentEsc", "responseScrollUp", "responseScrollDown", "responsePageUp",
		"responsePageDown", "toggleHelp", "toggleCodegen", "toggleTheme",
	}
	km := config.DefaultKeymap()
	for _, action := range actions {
		if _, ok := km[action]; !ok {
			t.Errorf("DefaultKeymap missing action %q", action)
		}
	}
}

// --- test helpers ---

func sidebarIndexOf(a *App, name string) int {
	for i, e := range a.sidebar {
		if e.name == name {
			return i
		}
	}
	return 0
}
