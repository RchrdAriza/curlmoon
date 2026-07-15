package tui

import (
	"curlmoon/internal/httpclient"
	"fmt"
	"strings"
	"time"

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
	methodsBarH  = 1
	statusBarH   = 1
)

var methods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

type sidebarEntry struct {
	name     string
	method   string
	url      string
	isFolder bool
	indent   int
	children []int
}

type Model struct {
	// Layout
	width  int
	height int
	ready  bool

	// Focus
	activePanel int

	// Sidebar
	sidebar    []sidebarEntry
	sidebarSel int
	sidebarOff int

	// Request
	urlInput    textinput.Model
	methodIndex int
	activeTab   int
	tabs        []string
	sending     bool

	// Response
	response   *httpclient.Response
	respView   viewport.Model
	respErr    error
	respReady  bool
	showResp   bool

	// Status
	statusMsg string
}

func NewModel() Model {
	ti := textinput.New()
	ti.Placeholder = "https://httpbin.org/get"
	ti.PromptStyle = lipgloss.NewStyle().Foreground(primary)
	ti.CharLimit = 2048
	ti.Width = 60
	ti.Focus()

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
		respView:    viewport.New(0, 0),
		statusMsg:   "Ready — Tab to switch panels, Ctrl+Enter to send",
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

type responseMsg struct {
	resp *httpclient.Response
	err  error
}

func (m Model) doRequest() tea.Msg {
	req := &httpclient.Request{
		Method: methods[m.methodIndex],
		URL:    m.urlInput.Value(),
	}
	if req.URL == "" {
		return responseMsg{err: fmt.Errorf("URL is empty")}
	}
	resp, err := httpclient.Execute(req)
	return responseMsg{resp: resp, err: err}
}

func (m Model) activeTabName() string {
	if m.activeTab < len(m.tabs) {
		return m.tabs[m.activeTab]
	}
	return ""
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
	return m
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.ready = true
		}
		m.respView.Width = m.width - sidebarWidth - 4
		m.respView.Height = m.height/2 - 3
		if m.respView.Height < 5 {
			m.respView.Height = 5
		}
		// Update URL input width
		urlWidth := m.width - sidebarWidth - 28
		if urlWidth < 20 {
			urlWidth = 20
		}
		m.urlInput.Width = urlWidth
		return m, nil

	case tea.KeyMsg:
		// Global keys
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "tab":
			m.activePanel = (m.activePanel + 1) % 3
			m.urlInput.Blur()
			if m.activePanel == panelRequest {
				m.urlInput.Focus()
			}
			m.statusMsg = fmt.Sprintf("Focus: %s", []string{"Sidebar", "Request", "Response"}[m.activePanel])
			return m, nil

		case "shift+tab":
			m.activePanel = (m.activePanel + 2) % 3
			m.urlInput.Blur()
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
		}

		// Panel-specific keys
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
			m.respView.SetContent(msg.resp.Body)
			m.respView.GotoTop()
			m.statusMsg = fmt.Sprintf("%d %s — %v — %d bytes",
				msg.resp.StatusCode, msg.resp.Status,
				msg.resp.Elapsed.Round(time.Millisecond),
				msg.resp.Size,
			)
		}
		return m, nil
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleSidebarKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.sidebarSel > 0 {
			m.sidebarSel--
			// Adjust scroll
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
	}
	return m, nil
}

func (m Model) handleRequestKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+r", "ctrl+enter":
		if !m.sending {
			m.sending = true
			m.statusMsg = "Sending request..."
			return m, func() tea.Msg { return m.doRequest() }
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
		// Change method with up/down when in request panel
		if msg.String() == "up" {
			m.methodIndex = (m.methodIndex - 1 + len(methods)) % len(methods)
		} else {
			m.methodIndex = (m.methodIndex + 1) % len(methods)
		}
		return m, nil
	}

	// URL input gets the key
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

	reqHeight := m.height/2 - 2
	respHeight := m.height/2 - 2
	if reqHeight < 5 {
		reqHeight = 5
	}
	if respHeight < 5 {
		respHeight = 5
	}

	sidebarView := m.renderSidebar(reqHeight + respHeight + statusBarH + 1)
	requestView := m.renderRequest()
	responseView := m.renderResponse()
	statusView := m.renderStatusBar()

	rightSide := lipgloss.JoinVertical(
		lipgloss.Top,
		requestView,
		statusView,
		responseView,
	)

	mainContent := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sidebarView,
		rightSide,
	)

	return mainContent
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
			// truncate long names
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
	return style.Height(height).Render(
		titleStyle.Render("Collections") + "\n" + content,
	)
}

func (m Model) renderRequest() string {
	style := panelBorder
	if m.activePanel == panelRequest {
		style = focusedPanelBorder
	}

	// Method selector + URL
	methodLabel := methods[m.methodIndex]
	methodBadge := methodStyle(methodLabel).Render(" " + methodLabel + " ")

	// URL input
	urlView := m.urlInput.View()

	// Method picker indicator
	methodPicker := lipgloss.NewStyle().Foreground(muted).Render("(↑↓ method)")

	topBar := lipgloss.JoinHorizontal(
		lipgloss.Center,
		methodBadge, lipgloss.NewStyle().Width(1).Render(""),
		urlView, lipgloss.NewStyle().Width(1).Render(""),
		methodPicker,
	)

	// Tabs
	var tabsView []string
	for i, tab := range m.tabs {
		if i == m.activeTab {
			tabsView = append(tabsView, activeTabStyle.Render(tab))
		} else {
			tabsView = append(tabsView, inactiveTabStyle.Render(tab))
		}
	}
	tabsRow := lipgloss.JoinHorizontal(lipgloss.Left, tabsView...)

	// Tab content (placeholder for now)
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

	// Send button
	sendBtn := "[Ctrl+Enter: Send]"
	if m.sending {
		sendBtn = "Sending..."
	}
	sendStyle := lipgloss.NewStyle().
		Foreground(primary).
		Bold(true).
		Padding(0, 1)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
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
	return lipgloss.NewStyle().
		Height(h).
		Padding(0, 2).
		Foreground(muted).
		Render("Headers editor coming in Phase 2\n\nFor now, headers are sent automatically.\nContent-Type: application/json will be added\nfor requests with body.")
}

func (m Model) renderBody() string {
	h := m.requestPanelHeight() - 7
	if h < 3 {
		h = 3
	}
	return lipgloss.NewStyle().
		Height(h).
		Padding(0, 2).
		Foreground(muted).
		Render("Body editor coming in Phase 2\n\nSupports: none, form-data,\nx-www-form-urlencoded, JSON, raw")
}

func (m Model) renderAuth() string {
	h := m.requestPanelHeight() - 7
	if h < 3 {
		h = 3
	}
	return lipgloss.NewStyle().
		Height(h).
		Padding(0, 2).
		Foreground(muted).
		Render("Auth helpers coming in Phase 4\n\nSupports: None, Basic, Bearer Token,\nAPI Key, OAuth 2.0")
}

func (m Model) renderParams() string {
	h := m.requestPanelHeight() - 7
	if h < 3 {
		h = 3
	}
	return lipgloss.NewStyle().
		Height(h).
		Padding(0, 2).
		Foreground(muted).
		Render("Query Params editor coming in Phase 4\n\nKey-value pairs that update the URL query string.")
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
			Render("Send a request to see the response here" + "\n\n" + "Try: Ctrl+Enter")
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

	// Response header bar
	statusText := statusColor(m.response.StatusCode).Render(
		fmt.Sprintf("%d %s", m.response.StatusCode, m.response.Status),
	)
	infoText := lipgloss.NewStyle().Foreground(muted).Render(
		fmt.Sprintf("%s  |  %d bytes",
			m.response.Elapsed.Round(time.Millisecond).String(),
			m.response.Size,
		),
	)

	// Response headers collapsible (show first few)
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

	// Body
	bodyView := m.respView.View()

	content := lipgloss.JoinVertical(
		lipgloss.Left,
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
	// Truncate if needed
	if len(statusText) > barWidth-2 {
		statusText = statusText[:barWidth-5] + "..."
	}

	return statusBarStyle.Width(barWidth).Render(statusText)
}
