package tui

import (
	"curlmoon/internal/collection"
	"curlmoon/internal/environment"
	"curlmoon/internal/history"
	"curlmoon/internal/httpclient"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	tabHeaders = 0
	tabBody    = 1
	tabAuth    = 2
	tabParams  = 3
)

const (
	panelSidebar  = "sidebar"
	panelURL      = "url"
	panelResponse = "response"
)

var methods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
var bodyTypes = []string{"none", "JSON", "raw", "form-data", "x-www-urlencoded"}
var tabNames = []string{"Headers", "Body", "Auth", "Params"}

type sidebarEntry struct {
	name     string
	method   string
	url      string
	isFolder bool
	indent   int
	collIdx  int   // index into App.collections; meaningful only when store != nil
	itemPath []int // path within collection.Item tree; empty means the entry is the collection root

	section string // "" (collection), "env", or "history" — which App slice this entry indexes into
	envIdx  int    // index into App.environments; meaningful when section == "env"
	histIdx int    // index into App.historyEntries; meaningful when section == "history"
}

// App holds all curlmoon state. It is plain, gocui-free data plus pure
// methods, so it can be exercised directly in tests without a real terminal.
type App struct {
	activePanel string // panelSidebar | panelURL | panelResponse
	subFocus    bool   // true when focus is inside the "content" view for activeTab

	sidebar    []sidebarEntry
	sidebarSel int
	sidebarOff int
	collapsed  map[string]bool // folder key -> collapsed, see sidebarFolderKey

	urlValue    string
	methodIndex int
	activeTab   int

	bodyType    int
	headersText string
	paramsText  string
	bodyText    string
	authText    string

	sending bool

	response *httpclient.Response
	respErr  error
	showResp bool

	statusMsg string

	store       *collection.Store
	collections []*collection.Collection

	envStore      *environment.Store
	environments  []*environment.Environment
	activeEnvName  string
	envEditIdx     int    // index into environments currently open in the content editor; -1 when not editing
	envEditText    string // live buffer for the environment being edited
	envEditPending bool   // true right after StartEnvEdit, until layout() has moved focus into "content"

	historyStore   *history.Store
	historyEntries []history.Entry

	promptMode   string // "", "newCollection", "newRequest", "rename", "confirmDelete", "newEnvironment", "renameEnv", "confirmDeleteEnv"
	promptTarget sidebarEntry
	promptText   string

	showHelp bool // true while the keybinding help overlay (Ctrl+/) is open
}

// NewApp builds a standalone app with the built-in example sidebar and no
// persistence backing (used mainly by tests).
func NewApp() *App {
	sidebar := []sidebarEntry{
		{name: "httpbin.org", isFolder: true, indent: 0},
		{name: "GET /get", method: "GET", url: "https://httpbin.org/get", indent: 1},
		{name: "POST /post", method: "POST", url: "https://httpbin.org/post", indent: 1},
		{name: "PUT /put", method: "PUT", url: "https://httpbin.org/put", indent: 1},
		{name: "DELETE /delete", method: "DELETE", url: "https://httpbin.org/delete", indent: 1},
		{name: "JSON Placeholder", isFolder: true, indent: 0},
		{name: "GET /todos/1", method: "GET", url: "https://jsonplaceholder.typicode.com/todos/1", indent: 1},
		{name: "GET /posts", method: "GET", url: "https://jsonplaceholder.typicode.com/posts", indent: 1},
		{name: "GitHub API", isFolder: true, indent: 0},
		{name: "GET /zen", method: "GET", url: "https://api.github.com/zen", indent: 1},
	}

	return &App{
		sidebar:     sidebar,
		activePanel: panelURL,
		activeTab:   tabHeaders,
		authText:    defaultAuthText,
		envEditIdx:  -1,
		collapsed:   make(map[string]bool),
		statusMsg:   "Ready — Tab switches panels, Enter to edit fields",
	}
}

// NewAppWithStore builds the real, persistence-backed app: collections are
// loaded from store (seeded with example collections on first run) and the
// last editor session is restored.
func NewAppWithStore(store *collection.Store) *App {
	a := NewApp()
	a.store = store

	cols, _ := store.LoadAll()
	a.collections = cols
	if len(a.collections) == 0 {
		a.seedDefaultCollections()
	}

	a.envStore = environment.NewStore(store.BaseDir)
	a.environments, _ = a.envStore.LoadAll()
	a.activeEnvName, _ = a.envStore.LoadActive()

	a.historyStore = history.NewStore(store.BaseDir)
	a.historyEntries, _ = a.historyStore.Load()

	a.rebuildSidebar()
	a.restoreSession()
	return a
}

func (a *App) seedDefaultCollections() {
	seed := []struct {
		name  string
		items []collection.Item
	}{
		{"httpbin.org", []collection.Item{
			collection.NewRequestItem("GET /get", "GET", "https://httpbin.org/get", nil, "", ""),
			collection.NewRequestItem("POST /post", "POST", "https://httpbin.org/post", nil, "", ""),
			collection.NewRequestItem("PUT /put", "PUT", "https://httpbin.org/put", nil, "", ""),
			collection.NewRequestItem("DELETE /delete", "DELETE", "https://httpbin.org/delete", nil, "", ""),
		}},
		{"JSON Placeholder", []collection.Item{
			collection.NewRequestItem("GET /todos/1", "GET", "https://jsonplaceholder.typicode.com/todos/1", nil, "", ""),
			collection.NewRequestItem("GET /posts", "GET", "https://jsonplaceholder.typicode.com/posts", nil, "", ""),
		}},
		{"GitHub API", []collection.Item{
			collection.NewRequestItem("GET /zen", "GET", "https://api.github.com/zen", nil, "", ""),
		}},
	}
	for _, s := range seed {
		c := collection.NewCollection(s.name)
		c.Item = s.items
		_ = a.store.Save(c)
		a.collections = append(a.collections, c)
	}
}

// sidebarFolderKey returns a stable identifier for a folder entry's
// collapsed/expanded state, independent of where it currently sits in the
// flattened sidebar slice (which gets rebuilt from scratch on every change).
func sidebarFolderKey(e sidebarEntry) string {
	switch e.section {
	case "env":
		return "env"
	case "history":
		return "history"
	default:
		return fmt.Sprintf("coll:%d:%v", e.collIdx, e.itemPath)
	}
}

// rebuildSidebar flattens the in-memory collections tree, plus environments
// and history when present, into sidebar rows. Children of a collapsed
// folder are omitted so the user can shrink the tree to reduce visual
// clutter; the folder row itself always stays visible.
func (a *App) rebuildSidebar() {
	var entries []sidebarEntry
	for ci, c := range a.collections {
		root := sidebarEntry{name: c.Info.Name, isFolder: true, collIdx: ci}
		entries = append(entries, root)
		if !a.collapsed[sidebarFolderKey(root)] {
			entries = append(entries, flattenItems(a, c.Item, ci, nil, 1)...)
		}
	}

	if a.envStore != nil {
		envRoot := sidebarEntry{name: "Environments", isFolder: true, section: "env"}
		entries = append(entries, envRoot)
		if !a.collapsed[sidebarFolderKey(envRoot)] {
			for i, env := range a.environments {
				name := env.Name
				if env.Name == a.activeEnvName {
					name = "● " + name
				}
				entries = append(entries, sidebarEntry{name: name, indent: 1, section: "env", envIdx: i})
			}
		}
	}

	if a.historyStore != nil {
		histRoot := sidebarEntry{name: "History", isFolder: true, section: "history"}
		entries = append(entries, histRoot)
		if !a.collapsed[sidebarFolderKey(histRoot)] {
			for i, h := range a.historyEntries {
				label := h.URL
				if h.StatusCode > 0 {
					label = fmt.Sprintf("%s (%d)", label, h.StatusCode)
				} else if h.Err != "" {
					label = label + " (error)"
				}
				entries = append(entries, sidebarEntry{name: label, method: h.Method, indent: 1, section: "history", histIdx: i})
			}
		}
	}

	a.sidebar = entries
	if a.sidebarSel >= len(a.sidebar) {
		a.sidebarSel = len(a.sidebar) - 1
	}
	if a.sidebarSel < 0 {
		a.sidebarSel = 0
	}
}

func flattenItems(a *App, items []collection.Item, collIdx int, parentPath []int, indent int) []sidebarEntry {
	var out []sidebarEntry
	for i, it := range items {
		path := append(append([]int{}, parentPath...), i)
		if it.IsFolder() {
			folder := sidebarEntry{name: it.Name, isFolder: true, indent: indent, collIdx: collIdx, itemPath: path}
			out = append(out, folder)
			if !a.collapsed[sidebarFolderKey(folder)] {
				out = append(out, flattenItems(a, it.Item, collIdx, path, indent+1)...)
			}
		} else {
			out = append(out, sidebarEntry{
				name: it.Name, method: it.Request.Method, url: it.Request.URL.Raw,
				indent: indent, collIdx: collIdx, itemPath: path,
			})
		}
	}
	return out
}

func (a *App) restoreSession() {
	if a.store == nil {
		return
	}
	sess, err := a.store.LoadSession()
	if err != nil || sess == nil {
		return
	}
	a.urlValue = sess.URL
	for i, meth := range methods {
		if meth == sess.Method {
			a.methodIndex = i
			break
		}
	}
	for i, bt := range bodyTypes {
		if bt == sess.BodyType {
			a.bodyType = i
			break
		}
	}
	a.bodyText = sess.Body
	if sess.ActiveTab >= 0 && sess.ActiveTab < len(tabNames) {
		a.activeTab = sess.ActiveTab
	}
	a.headersText = serializeKV(toKVPairs(sess.Headers))
	a.paramsText = serializeKV(toKVPairs(sess.Params))
	if sess.AuthText != "" {
		a.authText = sess.AuthText
	}
}

func (a *App) saveSession() {
	if a.store == nil {
		return
	}
	sess := &collection.Session{
		Method:    methods[a.methodIndex],
		URL:       a.urlValue,
		BodyType:  bodyTypes[a.bodyType],
		Body:      a.bodyText,
		AuthText:  a.authText,
		ActiveTab: a.activeTab,
	}
	for _, p := range parseKV(a.headersText) {
		sess.Headers = append(sess.Headers, collection.KeyVal{Key: p.Key, Value: p.Value})
	}
	for _, p := range parseKV(a.paramsText) {
		sess.Params = append(sess.Params, collection.KeyVal{Key: p.Key, Value: p.Value})
	}
	_ = a.store.SaveSession(sess)
}

func toKVPairs(kv []collection.KeyVal) []KeyValuePair {
	pairs := make([]KeyValuePair, len(kv))
	for i, h := range kv {
		pairs[i] = KeyValuePair{Key: h.Key, Value: h.Value}
	}
	return pairs
}

func (a *App) buildURL() string {
	base := a.urlValue
	pairs := parseKV(a.paramsText)
	if len(pairs) == 0 {
		return base
	}
	vals := url.Values{}
	for _, p := range pairs {
		if p.Key != "" {
			vals.Set(p.Key, p.Value)
		}
	}
	qs := vals.Encode()
	if qs == "" {
		return base
	}
	if strings.Contains(base, "?") {
		return base + "&" + qs
	}
	return base + "?" + qs
}

func (a *App) buildHeaders() map[string]string {
	h := kvToMap(parseKV(a.headersText))
	if a.bodyType == 1 {
		if _, ok := h["Content-Type"]; !ok {
			h["Content-Type"] = "application/json"
		}
	} else if a.bodyType == 2 {
		if _, ok := h["Content-Type"]; !ok {
			h["Content-Type"] = "text/plain"
		}
	} else if a.bodyType == 4 {
		if _, ok := h["Content-Type"]; !ok {
			h["Content-Type"] = "application/x-www-form-urlencoded"
		}
	}
	applyAuth(h, a.authText)
	return h
}

func (a *App) buildBody() string {
	switch a.bodyType {
	case 1, 2:
		return a.bodyText
	}
	return ""
}

// activeEnvVars returns the resolved variable map for the currently active
// environment, or an empty map if none is active.
func (a *App) activeEnvVars() map[string]string {
	for _, env := range a.environments {
		if env.Name == a.activeEnvName {
			return env.Vars()
		}
	}
	return nil
}

// doRequest executes the current request synchronously, resolving any
// {{variable}} tokens against the active environment first. Callers that
// need to stay responsive (the real gocui app) should run this in a
// goroutine and feed the result back via HandleResponse through
// *gocui.Gui.Execute.
func (a *App) doRequest() (*httpclient.Response, error) {
	vars := a.activeEnvVars()
	headers := a.buildHeaders()
	for k, v := range headers {
		headers[k] = environment.Resolve(v, vars)
	}
	req := &httpclient.Request{
		Method:   methods[a.methodIndex],
		URL:      environment.Resolve(a.buildURL(), vars),
		Headers:  headers,
		Body:     environment.Resolve(a.buildBody(), vars),
		BodyType: bodyTypes[a.bodyType],
	}
	if req.URL == "" {
		return nil, fmt.Errorf("URL is empty")
	}
	return httpclient.Execute(req)
}

// StartSending marks a request as in flight so the UI can show a spinner.
func (a *App) StartSending() {
	a.sending = true
	a.statusMsg = "Sending request..."
}

// HandleResponse applies the outcome of doRequest to the app state.
func (a *App) HandleResponse(resp *httpclient.Response, err error) {
	a.sending = false
	if err != nil {
		a.respErr = err
		a.showResp = false
		a.statusMsg = fmt.Sprintf("Error: %v", err)
		a.recordHistory(err.Error(), 0, "")
		return
	}
	a.response = resp
	a.respErr = nil
	a.showResp = true
	a.statusMsg = fmt.Sprintf("%d %s — %v — %d bytes",
		resp.StatusCode, resp.Status, resp.Elapsed, resp.Size)
	a.recordHistory("", resp.StatusCode, resp.Elapsed.String())
}

// recordHistory appends the just-executed request to the history log, keyed
// off the method/URL currently loaded in the editor.
func (a *App) recordHistory(errMsg string, statusCode int, elapsed string) {
	if a.historyStore == nil {
		return
	}
	entry := history.Entry{
		Method:     methods[a.methodIndex],
		URL:        a.urlValue,
		StatusCode: statusCode,
		Elapsed:    elapsed,
		Err:        errMsg,
		At:         time.Now(),
	}
	entries, err := a.historyStore.Add(entry)
	if err != nil {
		return
	}
	a.historyEntries = entries
	a.rebuildSidebar()
}

// CycleMethod moves the selected HTTP method by delta (wrapping around).
func (a *App) CycleMethod(delta int) {
	a.methodIndex = ((a.methodIndex+delta)%len(methods) + len(methods)) % len(methods)
}

// NextTab / PrevTab move the active request tab left/right.
func (a *App) NextTab() {
	if a.activeTab < len(tabNames)-1 {
		a.activeTab++
	}
}

func (a *App) PrevTab() {
	if a.activeTab > 0 {
		a.activeTab--
	}
}

// MoveSidebarSel moves the sidebar selection by delta, adjusting the scroll
// offset so the selection stays within [0, maxVisible) rows on screen.
func (a *App) MoveSidebarSel(delta int, maxVisible int) {
	if delta < 0 {
		if a.sidebarSel > 0 {
			a.sidebarSel--
			if a.sidebarSel < a.sidebarOff {
				a.sidebarOff = a.sidebarSel
			}
		}
		return
	}
	if a.sidebarSel < len(a.sidebar)-1 {
		a.sidebarSel++
		if maxVisible > 0 && a.sidebarSel-a.sidebarOff >= maxVisible {
			a.sidebarOff = a.sidebarSel - maxVisible + 1
		}
	}
}

// SelectSidebarEntry loads the currently selected sidebar request (if any)
// into the URL/method fields and focuses the URL panel. If the selected
// entry is a folder, it toggles that folder's collapsed state instead.
// Returns true if a request was loaded.
func (a *App) SelectSidebarEntry() bool {
	if len(a.sidebar) == 0 {
		return false
	}
	item := a.sidebar[a.sidebarSel]

	if item.isFolder {
		key := sidebarFolderKey(item)
		a.collapsed[key] = !a.collapsed[key]
		a.rebuildSidebar()
		return false
	}

	if item.section == "env" {
		a.toggleActiveEnvironment(item.envIdx)
		return false
	}
	if item.section == "history" {
		return a.loadHistoryEntry(item.histIdx)
	}

	if item.url == "" {
		return false
	}
	a.urlValue = item.url
	for i, meth := range methods {
		if meth == item.method {
			a.methodIndex = i
			break
		}
	}
	a.activePanel = panelURL
	a.statusMsg = fmt.Sprintf("Loaded: %s %s", item.method, item.url)
	return true
}

// EnterContentEditor moves focus into the tab content view (headers/body/
// auth/params).
func (a *App) EnterContentEditor() bool {
	a.subFocus = true
	a.statusMsg = "Esc to exit editor"
	return true
}

func (a *App) ExitContentEditor() {
	a.subFocus = false
	a.statusMsg = "Exited editor"
}

// toggleActiveEnvironment activates the environment at envIdx, or
// deactivates it if it's already the active one.
func (a *App) toggleActiveEnvironment(envIdx int) {
	if envIdx < 0 || envIdx >= len(a.environments) {
		return
	}
	env := a.environments[envIdx]
	if a.activeEnvName == env.Name {
		a.activeEnvName = ""
		a.statusMsg = fmt.Sprintf("Deactivated environment %q", env.Name)
	} else {
		a.activeEnvName = env.Name
		a.statusMsg = fmt.Sprintf("Activated environment %q", env.Name)
	}
	if a.envStore != nil {
		_ = a.envStore.SetActive(a.activeEnvName)
	}
	a.rebuildSidebar()
}

// loadHistoryEntry restores a past request's method/URL into the editor.
func (a *App) loadHistoryEntry(idx int) bool {
	if idx < 0 || idx >= len(a.historyEntries) {
		return false
	}
	h := a.historyEntries[idx]
	a.urlValue = h.URL
	for i, meth := range methods {
		if meth == h.Method {
			a.methodIndex = i
			break
		}
	}
	a.activePanel = panelURL
	a.statusMsg = fmt.Sprintf("Loaded from history: %s %s", h.Method, h.URL)
	return true
}

// StartEnvEdit opens the content editor pre-filled with the given
// environment's variables, formatted as "Key: Value" lines like headers.
func (a *App) StartEnvEdit(envIdx int) bool {
	if envIdx < 0 || envIdx >= len(a.environments) {
		return false
	}
	env := a.environments[envIdx]
	pairs := make([]KeyValuePair, len(env.Values))
	for i, kv := range env.Values {
		pairs[i] = KeyValuePair{Key: kv.Key, Value: kv.Value}
	}
	a.envEditIdx = envIdx
	a.envEditText = serializeKV(pairs)
	a.envEditPending = true
	a.subFocus = true
	a.statusMsg = fmt.Sprintf("Editing variables for %q — Esc to save", env.Name)
	return true
}

// SaveEnvEdit parses the in-progress environment edit text back into the
// environment's variables and persists it.
func (a *App) SaveEnvEdit() {
	if a.envEditIdx < 0 || a.envEditIdx >= len(a.environments) {
		a.envEditIdx = -1
		return
	}
	env := a.environments[a.envEditIdx]
	var values []environment.KeyVal
	for _, p := range parseKV(a.envEditText) {
		values = append(values, environment.KeyVal{Key: p.Key, Value: p.Value, Enabled: true})
	}
	env.Values = values
	if a.envStore != nil {
		_ = a.envStore.Save(env)
	}
	a.envEditIdx = -1
	a.envEditText = ""
	a.subFocus = false
	a.statusMsg = fmt.Sprintf("Saved variables for %q", env.Name)
}

// StartPrompt opens a sidebar prompt overlay (new collection/request, rename,
// delete confirmation).
func (a *App) StartPrompt(mode string, target sidebarEntry, prefill string) {
	a.promptMode = mode
	a.promptTarget = target
	a.promptText = prefill
}

func (a *App) CancelPrompt() {
	a.promptMode = ""
	a.promptText = ""
	a.statusMsg = "Cancelled"
}

// ConfirmPrompt applies the pending prompt action and closes the overlay.
func (a *App) ConfirmPrompt() {
	target := a.promptTarget
	switch a.promptMode {
	case "newCollection":
		name := strings.TrimSpace(a.promptText)
		if name != "" {
			if c, err := a.store.Create(name); err != nil {
				a.statusMsg = fmt.Sprintf("Error: %v", err)
			} else {
				a.collections = append(a.collections, c)
				a.statusMsg = fmt.Sprintf("Created collection %q", name)
			}
		}

	case "newRequest":
		name := strings.TrimSpace(a.promptText)
		if name != "" && target.collIdx < len(a.collections) {
			c := a.collections[target.collIdx]
			item := collection.NewRequestItem(name, methods[a.methodIndex], a.buildURL(), a.buildHeaders(), a.buildBody(), bodyTypes[a.bodyType])
			c.AddItemAt(nil, item)
			if err := a.store.Save(c); err != nil {
				a.statusMsg = fmt.Sprintf("Error: %v", err)
			} else {
				a.statusMsg = fmt.Sprintf("Saved request %q to %q", name, c.Info.Name)
			}
		}

	case "rename":
		newName := strings.TrimSpace(a.promptText)
		if newName != "" && target.collIdx < len(a.collections) {
			c := a.collections[target.collIdx]
			if len(target.itemPath) == 0 {
				if err := a.store.Rename(c.Info.Name, newName); err != nil {
					a.statusMsg = fmt.Sprintf("Error: %v", err)
				} else {
					c.Info.Name = newName
					a.statusMsg = fmt.Sprintf("Renamed to %q", newName)
				}
			} else {
				c.RenameItem(target.itemPath, newName)
				_ = a.store.Save(c)
				a.statusMsg = "Renamed"
			}
		}

	case "confirmDelete":
		if target.collIdx < len(a.collections) {
			c := a.collections[target.collIdx]
			if len(target.itemPath) == 0 {
				_ = a.store.Delete(c.Info.Name)
				a.collections = append(append([]*collection.Collection{}, a.collections[:target.collIdx]...), a.collections[target.collIdx+1:]...)
				a.statusMsg = "Collection deleted"
			} else {
				c.RemoveItem(target.itemPath)
				_ = a.store.Save(c)
				a.statusMsg = "Request deleted"
			}
		}

	case "newEnvironment":
		name := strings.TrimSpace(a.promptText)
		if name != "" && a.envStore != nil {
			if env, err := a.envStore.Create(name); err != nil {
				a.statusMsg = fmt.Sprintf("Error: %v", err)
			} else {
				a.environments = append(a.environments, env)
				a.statusMsg = fmt.Sprintf("Created environment %q", name)
			}
		}

	case "renameEnv":
		newName := strings.TrimSpace(a.promptText)
		if newName != "" && target.envIdx < len(a.environments) && a.envStore != nil {
			env := a.environments[target.envIdx]
			if err := a.envStore.Rename(env.Name, newName); err != nil {
				a.statusMsg = fmt.Sprintf("Error: %v", err)
			} else {
				if a.activeEnvName == env.Name {
					a.activeEnvName = newName
					_ = a.envStore.SetActive(newName)
				}
				env.Name = newName
				a.statusMsg = fmt.Sprintf("Renamed environment to %q", newName)
			}
		}

	case "confirmDeleteEnv":
		if target.envIdx < len(a.environments) && a.envStore != nil {
			env := a.environments[target.envIdx]
			_ = a.envStore.Delete(env.Name)
			if a.activeEnvName == env.Name {
				a.activeEnvName = ""
				_ = a.envStore.SetActive("")
			}
			a.environments = append(append([]*environment.Environment{}, a.environments[:target.envIdx]...), a.environments[target.envIdx+1:]...)
			a.statusMsg = "Environment deleted"
		}
	}

	a.promptMode = ""
	a.promptText = ""
	a.rebuildSidebar()
}
