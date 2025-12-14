package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/UPwith-me/Container-Maker/pkg/runtime"
)

// --- Color Palette (按用户参考图片) ---
var (
	colorPrimary   = lipgloss.Color("#E0AF68") // 橙黄色 (状态指示器)
	colorSecondary = lipgloss.Color("#7AA2F7") // 蓝色 (选中项高亮)
	colorSuccess   = lipgloss.Color("#E0AF68") // 橙黄色 (状态OK)
	colorError     = lipgloss.Color("#F7768E") // 红色 (状态错误)
	colorText      = lipgloss.Color("#C0CAF5") // 白色 (主文本)
	colorMuted     = lipgloss.Color("#565F89") // 灰色 (次要文本)
)

// ============================================================================
// PHASE 1: WELCOME SCREEN (Claude Code Style)
// ============================================================================

type WelcomeModel struct {
	ready bool
}

func (m WelcomeModel) Init() tea.Cmd {
	return nil
}

func (m WelcomeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", " ", "q", "ctrl+c":
			m.ready = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m WelcomeModel) View() string {
	if m.ready {
		return ""
	}

	// Welcome badge
	colorLavender := lipgloss.Color("#BB9AF7") // 淡紫色
	badge := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorLavender).
		Foreground(colorText).
		Padding(0, 1).
		Render("✽ Welcome to Container Maker")

	// ASCII Art Logo - CONTAINER MAKER (淡紫色)
	logo := lipgloss.NewStyle().Foreground(colorLavender).Render(`
  ██████╗ ██████╗ ███╗   ██╗████████╗ █████╗ ██╗███╗   ██╗███████╗██████╗ 
 ██╔════╝██╔═══██╗████╗  ██║╚══██╔══╝██╔══██╗██║████╗  ██║██╔════╝██╔══██╗
 ██║     ██║   ██║██╔██╗ ██║   ██║   ███████║██║██╔██╗ ██║█████╗  ██████╔╝
 ██║     ██║   ██║██║╚██╗██║   ██║   ██╔══██║██║██║╚██╗██║██╔══╝  ██╔══██╗
 ╚██████╗╚██████╔╝██║ ╚████║   ██║   ██║  ██║██║██║ ╚████║███████╗██║  ██║
  ╚═════╝ ╚═════╝ ╚═╝  ╚═══╝   ╚═╝   ╚═╝  ╚═╝╚═╝╚═╝  ╚═══╝╚══════╝╚═╝  ╚═╝
 ███╗   ███╗ █████╗ ██╗  ██╗███████╗██████╗ 
 ████╗ ████║██╔══██╗██║ ██╔╝██╔════╝██╔══██╗
 ██╔████╔██║███████║█████╔╝ █████╗  ██████╔╝
 ██║╚██╔╝██║██╔══██║██╔═██╗ ██╔══╝  ██╔══██╗
 ██║ ╚═╝ ██║██║  ██║██║  ██╗███████╗██║  ██║
 ╚═╝     ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝`)

	// Prompt
	prompt := lipgloss.NewStyle().Foreground(colorMuted).Render("Press ") +
		lipgloss.NewStyle().Foreground(colorSecondary).Underline(true).Render("Enter") +
		lipgloss.NewStyle().Foreground(colorMuted).Render(" to continue")

	return lipgloss.JoinVertical(lipgloss.Left,
		"",
		badge,
		logo,
		"",
		prompt,
		"",
	)
}

// RunWelcomeScreen shows the welcome screen
func RunWelcomeScreen() error {
	p := tea.NewProgram(WelcomeModel{})
	_, err := p.Run()
	return err
}

// ============================================================================
// PHASE 2: MAIN MENU
// ============================================================================

type MenuAction struct {
	Key         string
	Title       string
	Description string
	Handler     func() error
}

type HomeModel struct {
	cursor       int
	actions      []MenuAction
	status       *StatusInfo
	selected     int
	quitting     bool
	width        int
	height       int
	page         int // Current page (0-indexed)
	itemsPerPage int // Items per page
}

const defaultItemsPerPage = 6 // Show 6 items per page (1-6 keys)

func InitialHomeModel() HomeModel {
	return HomeModel{
		cursor:       0,
		page:         0,
		itemsPerPage: defaultItemsPerPage,
		actions: []MenuAction{
			// Page 1: Core Commands
			{"1", "Shell", "Enter dev container", func() error {
				return runExternalCommand("cm", "shell")
			}},
			{"2", "Clone", "Clone repo + auto-setup", func() error {
				fmt.Println("\nUsage: cm clone <repo-url>")
				fmt.Println("Example: cm clone https://github.com/user/repo")
				return nil
			}},
			{"3", "Init", "Initialize project", func() error {
				return runExternalCommand("cm", "init")
			}},
			{"4", "Code", "Open in VS Code", func() error {
				return runExternalCommand("cm", "code")
			}},
			{"5", "Templates", "Browse templates", func() error {
				return runExternalCommand("cm", "template")
			}},
			{"6", "Doctor", "Check environment", func() error {
				return runExternalCommand("cm", "doctor")
			}},
			// Page 2: More Commands
			{"1", "Share", "Generate share link", func() error {
				return runExternalCommand("cm", "share")
			}},
			{"2", "Config", "Manage settings", func() error {
				return runExternalCommand("cm", "config", "list")
			}},
			{"3", "Backend", "Container backends", func() error {
				return runExternalCommand("cm", "backend")
			}},
			{"4", "Remote", "Remote containers", func() error {
				return runExternalCommand("cm", "remote", "list")
			}},
			{"5", "Team", "Team settings", func() error {
				return runExternalCommand("cm", "team", "info")
			}},
			{"6", "Help", "Show help", func() error {
				return runExternalCommand("cm", "--help")
			}},
		},
		status: detectStatus(),
	}
}

func (m HomeModel) totalPages() int {
	return (len(m.actions) + m.itemsPerPage - 1) / m.itemsPerPage
}

func (m HomeModel) currentPageActions() []MenuAction {
	start := m.page * m.itemsPerPage
	end := start + m.itemsPerPage
	if end > len(m.actions) {
		end = len(m.actions)
	}
	return m.actions[start:end]
}

func runExternalCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (m HomeModel) Init() tea.Cmd {
	return nil
}

func (m HomeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			pageItems := len(m.currentPageActions())
			if m.cursor < pageItems-1 {
				m.cursor++
			}
		case "enter", " ":
			// Calculate actual index based on page
			actualIndex := m.page*m.itemsPerPage + m.cursor
			m.selected = actualIndex + 1
			return m, tea.Quit
		case "1", "2", "3", "4", "5", "6":
			num := int(msg.String()[0] - '0')
			pageActions := m.currentPageActions()
			if num <= len(pageActions) {
				m.selected = m.page*m.itemsPerPage + num
				return m, tea.Quit
			}
		case "left", "h":
			// Previous page
			if m.page > 0 {
				m.page--
				m.cursor = 0
			}
		case "right", "l":
			// Next page
			if m.page < m.totalPages()-1 {
				m.page++
				m.cursor = 0
			}
		case "tab":
			// Toggle page
			m.page = (m.page + 1) % m.totalPages()
			m.cursor = 0
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m HomeModel) View() string {
	if m.selected != 0 {
		return ""
	}
	if m.quitting {
		return "Goodbye! 👋\n"
	}

	var s strings.Builder

	// Header
	header := lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		Render("CONTAINER MAKER")

	s.WriteString(header + "\n\n")

	// Status Section (无截断，垂直列表)
	statusHeader := lipgloss.NewStyle().
		Foreground(colorMuted).
		Render("┌ System Status ────────────────────────────────────")
	s.WriteString(statusHeader + "\n")

	// Status items - 完整显示，不截断
	engineIcon := lipgloss.NewStyle().Foreground(colorSuccess).Render("●")
	configIcon := lipgloss.NewStyle().Foreground(colorSuccess).Render("●")
	gpuIcon := lipgloss.NewStyle().Foreground(colorSuccess).Render("●")

	if !m.status.HasBackend {
		engineIcon = lipgloss.NewStyle().Foreground(colorError).Render("●")
	}
	if !m.status.HasConfig {
		configIcon = lipgloss.NewStyle().Foreground(colorError).Render("●")
	}
	if !m.status.HasGPU {
		gpuIcon = lipgloss.NewStyle().Foreground(colorMuted).Render("○")
	}

	label := lipgloss.NewStyle().Foreground(colorMuted)
	value := lipgloss.NewStyle().Foreground(colorText)

	s.WriteString(fmt.Sprintf("│ %s %s %s\n", engineIcon, label.Render("Engine:"), value.Render(m.status.BackendStatus)))
	s.WriteString(fmt.Sprintf("│ %s %s %s\n", configIcon, label.Render("Config:"), value.Render(m.status.ProjectStatus)))
	s.WriteString(fmt.Sprintf("│ %s %s %s\n", gpuIcon, label.Render("GPU:   "), value.Render(m.status.GPUStatus)))

	statusFooter := lipgloss.NewStyle().
		Foreground(colorMuted).
		Render("└────────────────────────────────────────────────────")
	s.WriteString(statusFooter + "\n\n")

	// Menu
	menuHeader := lipgloss.NewStyle().
		Foreground(colorSecondary).
		Bold(true).
		Render("Operations")

	// Page indicator
	pageInfo := lipgloss.NewStyle().
		Foreground(colorMuted).
		Render(fmt.Sprintf(" (Page %d/%d)", m.page+1, m.totalPages()))

	s.WriteString(menuHeader + pageInfo + "\n\n")

	// Only show current page items
	pageActions := m.currentPageActions()
	for i, action := range pageActions {
		var row string
		if m.cursor == i {
			// Selected
			marker := lipgloss.NewStyle().Foreground(colorPrimary).Render("▸ ")
			key := lipgloss.NewStyle().Foreground(colorPrimary).Bold(true).Render(action.Key)
			title := lipgloss.NewStyle().Foreground(colorText).Bold(true).Render(action.Title)
			desc := lipgloss.NewStyle().Foreground(colorSecondary).Render(action.Description)
			row = fmt.Sprintf("%s[%s] %-12s %s", marker, key, title, desc)
		} else {
			// Normal
			marker := "  "
			key := lipgloss.NewStyle().Foreground(colorMuted).Render(action.Key)
			title := lipgloss.NewStyle().Foreground(colorMuted).Render(action.Title)
			desc := lipgloss.NewStyle().Foreground(colorMuted).Render(action.Description)
			row = fmt.Sprintf("%s[%s] %-12s %s", marker, key, title, desc)
		}
		s.WriteString(row + "\n")
	}

	// Footer with pagination hints
	s.WriteString("\n")
	footer := lipgloss.NewStyle().Foreground(colorMuted).Render("[1-6] Select • [←/→] Page • [Tab] Toggle • [q] Quit")
	s.WriteString(footer + "\n")

	return s.String()
}

// ============================================================================
// MAIN ENTRY POINT
// ============================================================================

func RunHomeScreen() error {
	// Phase 1: Show welcome screen
	if err := RunWelcomeScreen(); err != nil {
		return err
	}

	// Phase 2: Show main menu
	p := tea.NewProgram(InitialHomeModel())
	m, err := p.Run()
	if err != nil {
		return err
	}

	model, ok := m.(HomeModel)
	if !ok || model.selected == 0 {
		return nil
	}

	// Execute selected action
	if model.selected >= 1 && model.selected <= len(model.actions) {
		action := model.actions[model.selected-1]
		fmt.Println() // Newline before command output
		return action.Handler()
	}

	return nil
}

// ============================================================================
// STATUS DETECTION
// ============================================================================

type StatusInfo struct {
	BackendStatus string
	ProjectStatus string
	GPUStatus     string
	HasConfig     bool
	HasBackend    bool
	HasGPU        bool
}

func detectStatus() *StatusInfo {
	status := &StatusInfo{}

	// Detect backend
	detector := runtime.NewDetector()
	result := detector.Detect()

	if result.Active != nil {
		status.BackendStatus = fmt.Sprintf("%s v%s", result.Active.Name, result.Active.Version)
		status.HasBackend = true
	} else if len(result.Backends) > 0 {
		var names []string
		for _, b := range result.Backends {
			names = append(names, b.Name)
		}
		status.BackendStatus = fmt.Sprintf("Stopped (%s)", strings.Join(names, ", "))
		status.HasBackend = false
	} else {
		status.BackendStatus = "Not Installed"
		status.HasBackend = false
	}

	// Detect project config
	if _, err := os.Stat(".devcontainer/devcontainer.json"); err == nil {
		status.ProjectStatus = ".devcontainer/devcontainer.json"
		status.HasConfig = true
	} else if _, err := os.Stat("devcontainer.json"); err == nil {
		status.ProjectStatus = "devcontainer.json"
		status.HasConfig = true
	} else {
		status.ProjectStatus = "Not configured"
		status.HasConfig = false
	}

	// Detect GPU (完整显示，不截断)
	gpu := runtime.DetectGPU()
	if gpu.Available {
		status.GPUStatus = gpu.Name
		status.HasGPU = true
	} else {
		status.GPUStatus = "None detected"
		status.HasGPU = false
	}

	return status
}
