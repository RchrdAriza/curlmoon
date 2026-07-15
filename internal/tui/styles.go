package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	primary   = lipgloss.Color("#00BFFF") // DeepSkyBlue
	secondary = lipgloss.Color("#FFA500") // Orange
	success   = lipgloss.Color("#00FF7F") // SpringGreen
	errColor  = lipgloss.Color("#FF4444")
	muted     = lipgloss.Color("#888888")
	bgDark    = lipgloss.Color("#1A1A2E")
	bgMedium  = lipgloss.Color("#16213E")
	borderCol = lipgloss.Color("#0F3460")

	// Panel styles
	panelBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderCol).
			Padding(0, 1)

	focusedPanelBorder = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(primary).
				Padding(0, 1)

	siderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderCol).
			Padding(0, 1).Width(24)

	siderFocused = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primary).
			Padding(0, 1).Width(24)

	// Sidebar items
	itemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0E0E0")).
			Padding(0, 1)

	selectedItem = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(primary).
			Padding(0, 1)

	folderStyle = lipgloss.NewStyle().
			Foreground(primary).
			Bold(true).
			Padding(0, 1)

	// Method badges
	methodGet    = lipgloss.NewStyle().Foreground(success).Bold(true)
	methodPost   = lipgloss.NewStyle().Foreground(primary).Bold(true)
	methodPut    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500")).Bold(true)
	methodDelete = lipgloss.NewStyle().Foreground(errColor).Bold(true)
	methodPatch  = lipgloss.NewStyle().Foreground(lipgloss.Color("#BA55D3")).Bold(true)

	// Status bar
	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#0F3460")).
			Foreground(lipgloss.Color("#E0E0E0")).
			Padding(0, 1)

	// URL input
	urlInputStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(borderCol).
			Padding(0, 1)

	focusedUrlStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(primary).
			Padding(0, 1)

	// Labels
	titleStyle = lipgloss.NewStyle().
			Foreground(primary).
			Bold(true).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(muted).
			Padding(0, 1)

	// Response styles
	responseStatusSuccess = lipgloss.NewStyle().Foreground(success).Bold(true)
	responseStatusError   = lipgloss.NewStyle().Foreground(errColor).Bold(true)
	responseStatusOther   = lipgloss.NewStyle().Foreground(secondary).Bold(true)

	headerKeyStyle   = lipgloss.NewStyle().Foreground(primary).Bold(true)
	headerValueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#E0E0E0"))

	// Scrollbar
	scrollbarStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#0F3460"))

	// Tab styles
	activeTabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(primary).
			Padding(0, 2).
			Bold(true)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#E0E0E0")).
				Background(bgMedium).
				Padding(0, 2)
)

func methodStyle(method string) lipgloss.Style {
	switch method {
	case "GET":
		return methodGet
	case "POST":
		return methodPost
	case "PUT":
		return methodPut
	case "DELETE":
		return methodDelete
	case "PATCH":
		return methodPatch
	default:
		return methodGet
	}
}

func statusColor(statusCode int) lipgloss.Style {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return responseStatusSuccess
	case statusCode >= 400:
		return responseStatusError
	default:
		return responseStatusOther
	}
}
