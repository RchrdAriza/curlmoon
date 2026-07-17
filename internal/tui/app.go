package tui

import (
	"curlmoon/internal/collection"
	"curlmoon/internal/httpclient"
	"fmt"
	"net/url"
	"strings"
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
}

// App holds all curlmoon state. It is plain, gocui-free data plus pure
// methods, so it can be exercised directly in tests without a real terminal.
type App struct {
	activePanel string // panelSidebar | panelURL | panelResponse
	subFocus    bool   // true when focus is inside the "content" view for activeTab

	sidebar    []sidebarEntry
	sidebarSel int
	sidebarOff int

	urlValue    string
	methodIndex int
	activeTab   int

	bodyType    int
	headersText string
	paramsText  string
	bodyText    string

	sending bool

	response *httpclient.Response
	respErr  error
	showResp bool

	statusMsg string

	store       *collection.Store
	collections []*collection.Collection

	promptMode   string // "", "newCollection", "newRequest", "rename", "confirmDelete"
	promptTarget sidebarEntry
	promptText   string
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

// rebuildSidebar flattens the in-memory collections tree into sidebar rows.
func (a *App) rebuildSidebar() {
	var entries []sidebarEntry
	for ci, c := range a.collections {
		entries = append(entries, sidebarEntry{name: c.Info.Name, isFolder: true, collIdx: ci})
		entries = append(entries, flattenItems(c.Item, ci, nil, 1)...)
	}
	a.sidebar = entries
	if a.sidebarSel >= len(a.sidebar) {
		a.sidebarSel = len(a.sidebar) - 1
	}
	if a.sidebarSel < 0 {
		a.sidebarSel = 0
	}
}

func flattenItems(items []collection.Item, collIdx int, parentPath []int, indent int) []sidebarEntry {
	var out []sidebarEntry
	for i, it := range items {
		path := append(append([]int{}, parentPath...), i)
		if it.IsFolder() {
			out = append(out, sidebarEntry{name: it.Name, isFolder: true, indent: indent, collIdx: collIdx, itemPath: path})
			out = append(out, flattenItems(it.Item, collIdx, path, indent+1)...)
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
	return h
}

func (a *App) buildBody() string {
	switch a.bodyType {
	case 1, 2:
		return a.bodyText
	}
	return ""
}

// doRequest executes the current request synchronously. Callers that need to
// stay responsive (the real gocui app) should run this in a goroutine and
// feed the result back via HandleResponse through *gocui.Gui.Execute.
func (a *App) doRequest() (*httpclient.Response, error) {
	req := &httpclient.Request{
		Method:   methods[a.methodIndex],
		URL:      a.buildURL(),
		Headers:  a.buildHeaders(),
		Body:     a.buildBody(),
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
		return
	}
	a.response = resp
	a.respErr = nil
	a.showResp = true
	a.statusMsg = fmt.Sprintf("%d %s — %v — %d bytes",
		resp.StatusCode, resp.Status, resp.Elapsed, resp.Size)
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
// into the URL/method fields and focuses the URL panel. Returns true if a
// request was loaded.
func (a *App) SelectSidebarEntry() bool {
	if len(a.sidebar) == 0 {
		return false
	}
	item := a.sidebar[a.sidebarSel]
	if item.isFolder || item.url == "" {
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
// params), unless the active tab is the non-editable Auth placeholder.
func (a *App) EnterContentEditor() bool {
	if a.activeTab == tabAuth {
		return false
	}
	a.subFocus = true
	a.statusMsg = "Esc to exit editor"
	return true
}

func (a *App) ExitContentEditor() {
	a.subFocus = false
	a.statusMsg = "Exited editor"
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
	}

	a.promptMode = ""
	a.promptText = ""
	a.rebuildSidebar()
}
