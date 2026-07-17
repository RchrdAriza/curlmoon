package tui

import (
	"curlmoon/internal/collection"
	"curlmoon/internal/httpclient"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	panelSidebar  = 0
	panelRequest  = 1
	panelResponse = 2

	sidebarWidth = 26
)

var methods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
var bodyTypes = []string{"none", "JSON", "raw", "form-data", "x-www-urlencoded"}

type sidebarEntry struct {
	name     string
	method   string
	url      string
	isFolder bool
	indent   int
	collIdx  int   // index into Model.collections; meaningful only when store != nil
	itemPath []int // path within collection.Item tree; empty means the entry is the collection root
}

type Model struct {
	width  int
	height int
	ready  bool

	activePanel int

	sidebar    []sidebarEntry
	sidebarSel int
	sidebarOff int

	urlInput    textinput.Model
	methodIndex int
	activeTab   int
	tabs        []string
	sending     bool

	headers     KeyValueEditor
	bodyType    int
	bodyEditor  textarea.Model
	params      KeyValueEditor

	subFocus bool

	response  *httpclient.Response
	respView  viewport.Model
	respErr   error
	showResp  bool

	statusMsg string

	store       *collection.Store
	collections []*collection.Collection

	promptMode   string // "", "newCollection", "newRequest", "rename", "confirmDelete"
	promptInput  textinput.Model
	promptTarget sidebarEntry
}

func NewModel() Model {
	ti := textinput.New()
	ti.Placeholder = "https://httpbin.org/get"
	ti.PromptStyle = lipgloss.NewStyle().Foreground(primary)
	ti.CharLimit = 2048
	ti.Width = 60
	ti.Focus()

	headers := NewKeyValueEditor("Header", "Value")
	headers.SetWidth(60)
	headers.Blur()

	be := textarea.New()
	be.Placeholder = "Request body (JSON)..."
	be.CharLimit = 0
	be.SetWidth(50)
	be.SetHeight(8)
	be.ShowLineNumbers = false
	be.Blur()

	params := NewKeyValueEditor("Param", "Value")
	params.SetWidth(60)
	params.Blur()

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

	return Model{
		urlInput:    ti,
		methodIndex: 0,
		sidebar:     sidebar,
		sidebarSel:  0,
		activePanel: panelRequest,
		activeTab:   0,
		tabs:        []string{"Headers", "Body", "Auth", "Params"},
		headers:     *headers,
		bodyEditor:  be,
		params:      *params,
		respView:    viewport.New(0, 0),
		statusMsg:   "Ready — Tab switches panels, Enter to edit fields",
	}
}

// NewModelWithStore builds the real, persistence-backed app: collections are
// loaded from store (seeded with example collections on first run) and the
// last editor session is restored.
func NewModelWithStore(store *collection.Store) Model {
	m := NewModel()
	m.store = store

	cols, _ := store.LoadAll()
	m.collections = cols
	if len(m.collections) == 0 {
		m.seedDefaultCollections()
	}
	m.rebuildSidebar()
	m.restoreSession()
	return m
}

func (m *Model) seedDefaultCollections() {
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
		_ = m.store.Save(c)
		m.collections = append(m.collections, c)
	}
}

// rebuildSidebar flattens the in-memory collections tree into sidebar rows.
func (m *Model) rebuildSidebar() {
	var entries []sidebarEntry
	for ci, c := range m.collections {
		entries = append(entries, sidebarEntry{name: c.Info.Name, isFolder: true, collIdx: ci})
		entries = append(entries, flattenItems(c.Item, ci, nil, 1)...)
	}
	m.sidebar = entries
	if m.sidebarSel >= len(m.sidebar) {
		m.sidebarSel = len(m.sidebar) - 1
	}
	if m.sidebarSel < 0 {
		m.sidebarSel = 0
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

func newPromptInput(placeholder string) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 128
	ti.Width = 40
	ti.Focus()
	return ti
}

func (m *Model) restoreSession() {
	if m.store == nil {
		return
	}
	sess, err := m.store.LoadSession()
	if err != nil || sess == nil {
		return
	}
	m.urlInput.SetValue(sess.URL)
	for i, meth := range methods {
		if meth == sess.Method {
			m.methodIndex = i
			break
		}
	}
	for i, bt := range bodyTypes {
		if bt == sess.BodyType {
			m.bodyType = i
			break
		}
	}
	m.bodyEditor.SetValue(sess.Body)
	if sess.ActiveTab >= 0 && sess.ActiveTab < len(m.tabs) {
		m.activeTab = sess.ActiveTab
	}
	m.headers = *NewKeyValueEditorWithPairs("Header", "Value", toKVPairs(sess.Headers))
	m.params = *NewKeyValueEditorWithPairs("Param", "Value", toKVPairs(sess.Params))
}

func (m Model) saveSession() {
	if m.store == nil {
		return
	}
	sess := &collection.Session{
		Method:    methods[m.methodIndex],
		URL:       m.urlInput.Value(),
		BodyType:  bodyTypes[m.bodyType],
		Body:      m.bodyEditor.Value(),
		ActiveTab: m.activeTab,
	}
	for _, p := range m.headers.Pairs() {
		sess.Headers = append(sess.Headers, collection.KeyVal{Key: p.Key, Value: p.Value})
	}
	for _, p := range m.params.Pairs() {
		sess.Params = append(sess.Params, collection.KeyVal{Key: p.Key, Value: p.Value})
	}
	_ = m.store.SaveSession(sess)
}

func toKVPairs(kv []collection.KeyVal) []KeyValuePair {
	pairs := make([]KeyValuePair, len(kv))
	for i, h := range kv {
		pairs[i] = KeyValuePair{Key: h.Key, Value: h.Value}
	}
	return pairs
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, textarea.Blink)
}

type responseMsg struct {
	resp *httpclient.Response
	err  error
}

func (m Model) buildURL() string {
	base := m.urlInput.Value()
	pairs := m.params.Pairs()
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

func (m Model) buildHeaders() map[string]string {
	h := m.headers.ToMap()
	if m.bodyType == 1 {
		if _, ok := h["Content-Type"]; !ok {
			h["Content-Type"] = "application/json"
		}
	} else if m.bodyType == 2 {
		if _, ok := h["Content-Type"]; !ok {
			h["Content-Type"] = "text/plain"
		}
	} else if m.bodyType == 4 {
		if _, ok := h["Content-Type"]; !ok {
			h["Content-Type"] = "application/x-www-form-urlencoded"
		}
	}
	return h
}

func (m Model) buildBody() string {
	switch m.bodyType {
	case 1:
		return m.bodyEditor.Value()
	case 2:
		return m.bodyEditor.Value()
	}
	return ""
}

func (m Model) doRequest() tea.Msg {
	req := &httpclient.Request{
		Method:   methods[m.methodIndex],
		URL:      m.buildURL(),
		Headers:  m.buildHeaders(),
		Body:     m.buildBody(),
		BodyType: bodyTypes[m.bodyType],
	}
	if req.URL == "" {
		return responseMsg{err: fmt.Errorf("URL is empty")}
	}
	resp, err := httpclient.Execute(req)
	return responseMsg{resp: resp, err: err}
}

func (m Model) initLayout(w, h int) Model {
	m.width = w
	m.height = h
	m.ready = true
	m.respView.Width = w - sidebarWidth - 4
	m.respView.Height = h/2 - 3
	if m.respView.Height < 5 {
		m.respView.Height = 5
	}
	urlWidth := w - sidebarWidth - 28
	if urlWidth < 20 {
		urlWidth = 20
	}
	m.urlInput.Width = urlWidth

	editorW := w - sidebarWidth - 10
	if editorW < 40 {
		editorW = 40
	}
	m.headers.SetWidth(editorW)
	m.params.SetWidth(editorW)
	m.bodyEditor.SetWidth(editorW - 4)
	bodyH := m.height/2 - 9
	if bodyH < 3 {
		bodyH = 3
	}
	m.bodyEditor.SetHeight(bodyH)
	return m
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.initLayout(msg.Width, msg.Height), nil

	case tea.KeyMsg:
		if m.promptMode != "" {
			return m.handlePromptKey(msg)
		}

		switch msg.String() {
		case "ctrl+c", "q":
			m.saveSession()
			return m, tea.Quit

		case "tab":
			if m.activePanel == panelRequest && m.subFocus {
				return m.handleRequestKey(msg)
			}
			m.activePanel = (m.activePanel + 1) % 3
			m.urlInput.Blur()
			m.headers.Blur()
			m.params.Blur()
			m.bodyEditor.Blur()
			m.subFocus = false
			if m.activePanel == panelRequest {
				m.urlInput.Focus()
			}
			m.statusMsg = fmt.Sprintf("Focus: %s", []string{"Sidebar", "Request", "Response"}[m.activePanel])
			return m, nil

		case "shift+tab":
			if m.activePanel == panelRequest && m.subFocus {
				return m.handleRequestKey(msg)
			}
			m.activePanel = (m.activePanel + 2) % 3
			m.urlInput.Blur()
			m.headers.Blur()
			m.params.Blur()
			m.bodyEditor.Blur()
			m.subFocus = false
			if m.activePanel == panelRequest {
				m.urlInput.Focus()
			}
			m.statusMsg = fmt.Sprintf("Focus: %s", []string{"Sidebar", "Request", "Response"}[m.activePanel])
			return m, nil

		case "enter":
			if m.activePanel == panelSidebar {
				item := m.sidebar[m.sidebarSel]
				if !item.isFolder && item.url != "" {
					m.urlInput.SetValue(item.url)
					for i, meth := range methods {
						if meth == item.method {
							m.methodIndex = i
							break
						}
					}
					m.activePanel = panelRequest
					m.urlInput.Focus()
					m.statusMsg = fmt.Sprintf("Loaded: %s %s", item.method, item.url)
				}
				return m, nil
			}
			if m.activePanel == panelRequest && !m.subFocus {
				if m.activeTab != 2 {
					m.subFocus = true
					m.urlInput.Blur()
					switch m.activeTab {
					case 0:
						m.headers.Focus()
					case 1:
						m.bodyEditor.Focus()
					case 3:
						m.params.Focus()
					}
					m.statusMsg = "Tab/S-Tab cycles fields, Esc to exit editor"
					return m, nil
				}
			}
		case "esc":
			if m.activePanel == panelRequest && m.subFocus {
				m.subFocus = false
				m.urlInput.Focus()
				m.headers.Blur()
				m.params.Blur()
				m.bodyEditor.Blur()
				m.statusMsg = "Exited editor"
				return m, nil
			}
		}

		switch m.activePanel {
		case panelSidebar:
			return m.handleSidebarKey(msg)
		case panelRequest:
			return m.handleRequestKey(msg)
		case panelResponse:
			return m.handleResponseKey(msg)
		}

	case responseMsg:
		m.sending = false
		if msg.err != nil {
			m.respErr = msg.err
			m.showResp = false
			m.statusMsg = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.response = msg.resp
			m.respErr = nil
			m.showResp = true
			colored := highlightJSON(msg.resp.Body)
			m.respView.SetContent(colored)
			m.respView.GotoTop()
			m.statusMsg = fmt.Sprintf("%d %s — %v — %d bytes",
				msg.resp.StatusCode, msg.resp.Status,
				msg.resp.Elapsed.Round(time.Millisecond),
				msg.resp.Size,
			)
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handleSidebarKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.sidebarSel > 0 {
			m.sidebarSel--
			if m.sidebarSel < m.sidebarOff {
				m.sidebarOff = m.sidebarSel
			}
		}
	case "down", "j":
		if m.sidebarSel < len(m.sidebar)-1 {
			m.sidebarSel++
			maxVisible := m.requestPanelHeight()
			if m.sidebarSel-m.sidebarOff >= maxVisible {
				m.sidebarOff = m.sidebarSel - maxVisible + 1
			}
		}

	case "n":
		if m.store != nil {
			m.promptMode = "newCollection"
			m.promptInput = newPromptInput("Collection name")
		}

	case "a":
		if m.store != nil && len(m.sidebar) > 0 {
			m.promptTarget = sidebarEntry{collIdx: m.sidebar[m.sidebarSel].collIdx}
			m.promptMode = "newRequest"
			m.promptInput = newPromptInput("Request name")
		}

	case "r":
		if m.store != nil && len(m.sidebar) > 0 {
			sel := m.sidebar[m.sidebarSel]
			m.promptTarget = sel
			m.promptMode = "rename"
			m.promptInput = newPromptInput("New name")
			m.promptInput.SetValue(sel.name)
		}

	case "d":
		if m.store != nil && len(m.sidebar) > 0 {
			sel := m.sidebar[m.sidebarSel]
			m.promptTarget = sel
			m.promptMode = "confirmDelete"
			m.promptInput = textinput.Model{}
		}
	}
	return m, nil
}

// handlePromptKey routes key input while a sidebar prompt overlay (new
// collection/request, rename, delete confirmation) is active.
func (m Model) handlePromptKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.promptMode == "confirmDelete" {
		switch msg.String() {
		case "y":
			return m.confirmPrompt()
		default:
			m.promptMode = ""
			m.statusMsg = "Cancelled"
		}
		return m, nil
	}

	switch msg.String() {
	case "esc":
		m.promptMode = ""
		return m, nil
	case "enter":
		return m.confirmPrompt()
	}

	var cmd tea.Cmd
	m.promptInput, cmd = m.promptInput.Update(msg)
	return m, cmd
}

func (m Model) confirmPrompt() (tea.Model, tea.Cmd) {
	target := m.promptTarget
	switch m.promptMode {
	case "newCollection":
		name := strings.TrimSpace(m.promptInput.Value())
		if name != "" {
			if c, err := m.store.Create(name); err != nil {
				m.statusMsg = fmt.Sprintf("Error: %v", err)
			} else {
				m.collections = append(m.collections, c)
				m.statusMsg = fmt.Sprintf("Created collection %q", name)
			}
		}

	case "newRequest":
		name := strings.TrimSpace(m.promptInput.Value())
		if name != "" && target.collIdx < len(m.collections) {
			c := m.collections[target.collIdx]
			item := collection.NewRequestItem(name, methods[m.methodIndex], m.buildURL(), m.buildHeaders(), m.buildBody(), bodyTypes[m.bodyType])
			c.AddItemAt(nil, item)
			if err := m.store.Save(c); err != nil {
				m.statusMsg = fmt.Sprintf("Error: %v", err)
			} else {
				m.statusMsg = fmt.Sprintf("Saved request %q to %q", name, c.Info.Name)
			}
		}

	case "rename":
		newName := strings.TrimSpace(m.promptInput.Value())
		if newName != "" && target.collIdx < len(m.collections) {
			c := m.collections[target.collIdx]
			if len(target.itemPath) == 0 {
				if err := m.store.Rename(c.Info.Name, newName); err != nil {
					m.statusMsg = fmt.Sprintf("Error: %v", err)
				} else {
					c.Info.Name = newName
					m.statusMsg = fmt.Sprintf("Renamed to %q", newName)
				}
			} else {
				c.RenameItem(target.itemPath, newName)
				_ = m.store.Save(c)
				m.statusMsg = "Renamed"
			}
		}

	case "confirmDelete":
		if target.collIdx < len(m.collections) {
			c := m.collections[target.collIdx]
			if len(target.itemPath) == 0 {
				_ = m.store.Delete(c.Info.Name)
				m.collections = append(append([]*collection.Collection{}, m.collections[:target.collIdx]...), m.collections[target.collIdx+1:]...)
				m.statusMsg = "Collection deleted"
			} else {
				c.RemoveItem(target.itemPath)
				_ = m.store.Save(c)
				m.statusMsg = "Request deleted"
			}
		}
	}

	m.promptMode = ""
	m.rebuildSidebar()
	return m, nil
}

func (m Model) handleRequestKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.subFocus {
		switch msg.String() {
		case "ctrl+r", "ctrl+enter":
			if !m.sending {
				m.sending = true
				m.statusMsg = "Sending request..."
				return m, func() tea.Msg { return m.doRequest() }
			}
			return m, nil
		case "esc":
			m.subFocus = false
			m.urlInput.Focus()
			m.headers.Blur()
			m.params.Blur()
			m.bodyEditor.Blur()
			m.statusMsg = "Exited editor"
			return m, nil
		}

		switch m.activeTab {
		case 0:
			m.headers.HandleKey(msg)
		case 1:
			var cmd tea.Cmd
			m.bodyEditor, cmd = m.bodyEditor.Update(msg)
			_ = cmd
		case 3:
			m.params.HandleKey(msg)
		}
		return m, nil
	}

	switch msg.String() {
	case "ctrl+r", "ctrl+enter":
		if !m.sending {
			m.sending = true
			m.statusMsg = "Sending request..."
			return m, func() tea.Msg { return m.doRequest() }
		}
		return m, nil

	case "tab", "shift+tab":
		if m.activeTab == 1 || m.activeTab == 2 {
			break
		}
		m.subFocus = true
		m.urlInput.Blur()
		switch m.activeTab {
		case 0:
			m.headers.Focus()
			m.headers.HandleKey(msg)
		case 3:
			m.params.Focus()
			m.params.HandleKey(msg)
		}
		m.statusMsg = "Tab/S-Tab cycles fields, Esc to exit"
		return m, nil

	case "enter":
		if m.activeTab != 2 {
			m.subFocus = true
			m.urlInput.Blur()
			switch m.activeTab {
			case 0:
				m.headers.Focus()
			case 1:
				m.bodyEditor.Focus()
			case 3:
				m.params.Focus()
			}
			m.statusMsg = "Tab/S-Tab cycles fields, Esc to exit"
		}
		return m, nil

	case "left":
		if m.activeTab > 0 {
			m.activeTab--
		}
		return m, nil

	case "right":
		if m.activeTab < len(m.tabs)-1 {
			m.activeTab++
		}
		return m, nil

	case "up", "down":
		if msg.String() == "up" {
			m.methodIndex = (m.methodIndex - 1 + len(methods)) % len(methods)
		} else {
			m.methodIndex = (m.methodIndex + 1) % len(methods)
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.urlInput, cmd = m.urlInput.Update(msg)
	return m, cmd
}

func (m Model) handleResponseKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.respView, cmd = m.respView.Update(msg)
	return m, cmd
}

func (m Model) requestPanelHeight() int {
	return m.height/2 - 2
}

func (m Model) responsePanelHeight() int {
	h := m.height/2 - 2
	if h < 5 {
		h = 5
	}
	return h
}

func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	sidebarView := m.renderSidebar(m.height - 2)
	requestView := m.renderRequest()
	responseView := m.renderResponse()
	statusView := m.renderStatusBar()

	rightSide := lipgloss.JoinVertical(lipgloss.Top, requestView, statusView, responseView)
	view := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, rightSide)
	if m.promptMode != "" {
		view = lipgloss.JoinVertical(lipgloss.Top, view, m.renderPrompt())
	}
	return view
}

func (m Model) renderPrompt() string {
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(secondary).Padding(0, 1)
	switch m.promptMode {
	case "newCollection":
		return box.Render("New collection name: " + m.promptInput.View())
	case "newRequest":
		return box.Render("Save as (request name): " + m.promptInput.View())
	case "rename":
		return box.Render("Rename to: " + m.promptInput.View())
	case "confirmDelete":
		return box.Render(fmt.Sprintf("Delete %q? (y/n)", m.promptTarget.name))
	}
	return ""
}

func (m Model) renderSidebar(height int) string {
	style := siderStyle
	if m.activePanel == panelSidebar {
		style = siderFocused
	}

	maxItems := height - 3
	if maxItems < 1 {
		maxItems = 1
	}

	var items []string
	visible := m.sidebar
	if m.sidebarOff > 0 {
		visible = visible[m.sidebarOff:]
	}
	for i, item := range visible {
		if i >= maxItems {
			break
		}
		idx := m.sidebarOff + i
		prefix := ""
		if item.indent > 0 {
			prefix = strings.Repeat("  ", item.indent)
		}
		if item.isFolder {
			line := prefix + "📁 " + item.name
			if idx == m.sidebarSel {
				items = append(items, selectedItem.Render(line))
			} else {
				items = append(items, folderStyle.Render(line))
			}
		} else {
			meth := methodStyle(item.method).Render(item.method)
			display := item.name
			if len(display) > 16 {
				display = display[:14] + ".."
			}
			line := fmt.Sprintf("%s%s %s", prefix, meth, display)
			if idx == m.sidebarSel {
				items = append(items, selectedItem.Render(line))
			} else {
				items = append(items, itemStyle.Render(line))
			}
		}
	}

	content := lipgloss.NewStyle().Width(22).Render(strings.Join(items, "\n"))
	return style.Height(height).Render(titleStyle.Render("Collections") + "\n" + content)
}

func (m Model) renderRequest() string {
	style := panelBorder
	if m.activePanel == panelRequest {
		style = focusedPanelBorder
	}

	methodLabel := methods[m.methodIndex]
	methodBadge := methodStyle(methodLabel).Render(" " + methodLabel + " ")
	urlView := m.urlInput.View()
	methodPicker := lipgloss.NewStyle().Foreground(muted).Render("(↑↓ method)")

	topBar := lipgloss.JoinHorizontal(lipgloss.Center,
		methodBadge, lipgloss.NewStyle().Width(1).Render(""),
		urlView, lipgloss.NewStyle().Width(1).Render(""),
		methodPicker,
	)

	var tabsView []string
	for i, tab := range m.tabs {
		if i == m.activeTab {
			tabsView = append(tabsView, activeTabStyle.Render(tab))
		} else {
			tabsView = append(tabsView, inactiveTabStyle.Render(tab))
		}
	}
	tabsRow := lipgloss.JoinHorizontal(lipgloss.Left, tabsView...)

	tabContent := ""
	switch m.activeTab {
	case 0:
		tabContent = m.renderHeaders()
	case 1:
		tabContent = m.renderBody()
	case 2:
		tabContent = m.renderAuth()
	case 3:
		tabContent = m.renderParams()
	}

	sendBtn := "[Ctrl+R: Send]"
	if m.sending {
		sendBtn = "Sending...  "
		if time.Now().UnixMilli()%2000 < 1000 {
			sendBtn = "Sending... "
		}
	}
	sendStyle := lipgloss.NewStyle().Foreground(primary).Bold(true).Padding(0, 1)

	content := lipgloss.JoinVertical(lipgloss.Left,
		topBar,
		lipgloss.NewStyle().Height(1).Render(""),
		tabsRow,
		tabContent,
		lipgloss.NewStyle().Height(1).Render(""),
		sendStyle.Render(sendBtn),
	)
	return style.Render(content)
}

func (m Model) renderHeaders() string {
	h := m.requestPanelHeight() - 7
	if h < 3 {
		h = 3
	}
	if !m.subFocus || m.activeTab != 0 {
		return lipgloss.NewStyle().Height(h).Padding(0, 1).Render(
			m.headers.View(h),
		)
	}
	return lipgloss.NewStyle().Height(h).Padding(0, 1).Render(m.headers.View(h))
}

func (m Model) renderBody() string {
	h := m.requestPanelHeight() - 8
	if h < 3 {
		h = 3
	}

	var subTabs []string
	for i, bt := range bodyTypes {
		if i == m.bodyType {
			subTabs = append(subTabs, activeTabStyle.Render(bt))
		} else {
			subTabs = append(subTabs, inactiveTabStyle.Render(bt))
		}
	}
	subRow := lipgloss.JoinHorizontal(lipgloss.Left, subTabs...)

	var editorView string
	switch m.bodyType {
	case 0:
		editorView = lipgloss.NewStyle().Height(h).Padding(0, 2).Foreground(muted).
			Render("No body. Select JSON or raw above.")
	case 1, 2:
		m.bodyEditor.SetHeight(h - 4)
		if h-4 < 2 {
			m.bodyEditor.SetHeight(2)
		}
		editorView = m.bodyEditor.View()
	case 3:
		editorView = "Form-data editor (coming soon)"
	case 4:
		editorView = "URL-encoded editor (coming soon)"
	}

	return lipgloss.JoinVertical(lipgloss.Left, subRow, editorView)
}

func (m Model) renderAuth() string {
	h := m.requestPanelHeight() - 7
	if h < 3 {
		h = 3
	}
	return lipgloss.NewStyle().Height(h).Padding(0, 2).Foreground(muted).
		Render("Auth helpers coming in Phase 4" + "\n\n" +
			"Supports: None, Basic, Bearer Token," + "\n" +
			"API Key, OAuth 2.0")
}

func (m Model) renderParams() string {
	h := m.requestPanelHeight() - 7
	if h < 3 {
		h = 3
	}
	return lipgloss.NewStyle().Height(h).Padding(0, 1).Render(m.params.View(h))
}

func (m Model) renderResponse() string {
	style := panelBorder
	if m.activePanel == panelResponse {
		style = focusedPanelBorder
	}

	if !m.showResp && m.respErr == nil {
		msg := lipgloss.NewStyle().
			Foreground(muted).
			Width(m.width - sidebarWidth - 6).
			Height(m.responsePanelHeight() - 2).
			Render("Send a request to see the response here" + "\n\n" + "Try: Ctrl+R")
		return style.Render(msg)
	}

	if m.respErr != nil {
		msg := lipgloss.NewStyle().
			Foreground(errColor).
			Width(m.width - sidebarWidth - 6).
			Height(m.responsePanelHeight() - 2).
			Render("Error:" + "\n" + fmt.Sprintf("%v", m.respErr))
		return style.Render(msg)
	}

	if m.response == nil {
		return style.Render("")
	}

	statusText := statusColor(m.response.StatusCode).Render(
		fmt.Sprintf("%d %s", m.response.StatusCode, m.response.Status),
	)
	infoText := lipgloss.NewStyle().Foreground(muted).Render(
		fmt.Sprintf("%s  |  %d bytes",
			m.response.Elapsed.Round(time.Millisecond).String(),
			m.response.Size,
		),
	)

	headerLines := strings.Split(m.response.HeaderStr, "\n")
	headerPreview := ""
	maxHeaderLines := 4
	for i, line := range headerLines {
		if i >= maxHeaderLines {
			headerPreview += "...\n"
			break
		}
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) == 2 {
			headerPreview += headerKeyStyle.Render(parts[0]) + ": " + headerValueStyle.Render(parts[1]) + "\n"
		} else {
			headerPreview += line + "\n"
		}
	}

	bodyView := m.respView.View()

	content := lipgloss.JoinVertical(lipgloss.Left,
		statusText+"  "+infoText,
		headerPreview,
		bodyView,
	)
	return style.Render(content)
}

func (m Model) renderStatusBar() string {
	statusText := m.statusMsg
	if m.sending {
		spinner := "⏳ Sending... "
		if time.Now().UnixMilli()%2000 < 1000 {
			spinner = "⏳ Sending...  "
		}
		statusText = spinner
	}

	barWidth := m.width - sidebarWidth - 4
	if barWidth < 10 {
		barWidth = 10
	}
	if len(statusText) > barWidth-2 {
		statusText = statusText[:barWidth-5] + "..."
	}
	return statusBarStyle.Width(barWidth).Render(statusText)
}
