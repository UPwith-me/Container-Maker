package monitor

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/docker/docker/api/types/container"
)

// Dashboard style constants
var (
	// Colors
	colorPrimary   = lipgloss.Color("#7C3AED") // Purple
	colorSecondary = lipgloss.Color("#06B6D4") // Cyan
	colorSuccess   = lipgloss.Color("#10B981") // Green
	colorWarning   = lipgloss.Color("#F59E0B") // Yellow
	colorDanger    = lipgloss.Color("#EF4444") // Red
	colorMuted     = lipgloss.Color("#6B7280") // Gray

	// Styles
	titleStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	selectedStyle = lipgloss.NewStyle().
			Background(colorPrimary).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1)

	normalStyle = lipgloss.NewStyle().
			Padding(0, 1)

	runningStyle = lipgloss.NewStyle().
			Foreground(colorSuccess)

	stoppedStyle = lipgloss.NewStyle().
			Foreground(colorDanger)

	mutedStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 1)
)

// DashboardModel is the Bubble Tea model for the monitoring dashboard
type DashboardModel struct {
	collector   *DockerCollector
	containers  []*ContainerInfo
	metrics     map[string]*ContainerMetrics
	events      []*ContainerEvent
	cursor      int
	width       int
	height      int
	showLogs    bool
	spinner     spinner.Model
	loading     bool
	err         error
	lastRefresh time.Time
	refreshRate time.Duration
	ctx         context.Context
	cancel      context.CancelFunc
}

// Bubble Tea messages
type tickMsg time.Time
type containerListMsg []*ContainerInfo
type metricsUpdateMsg map[string]*ContainerMetrics
type eventMsg *ContainerEvent
type errMsg struct{ err error }

// NewDashboardModel creates a new dashboard model
func NewDashboardModel() (*DashboardModel, error) {
	collector, err := NewDockerCollector()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colorPrimary)

	return &DashboardModel{
		collector:   collector,
		metrics:     make(map[string]*ContainerMetrics),
		spinner:     s,
		loading:     true,
		refreshRate: 2 * time.Second,
		ctx:         ctx,
		cancel:      cancel,
	}, nil
}

// Init initializes the model
func (m DashboardModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.loadContainers,
		m.tickCmd(),
	)
}

// Update handles messages
func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.cancel()
			_ = m.collector.Close()
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.containers)-1 {
				m.cursor++
			}

		case "r":
			m.loading = true
			return m, m.loadContainers

		case "l":
			m.showLogs = !m.showLogs
			if m.showLogs && len(m.containers) > 0 {
				return m, m.startLogs(m.containers[m.cursor].ID)
			}

		case "s":
			// Start selected container
			if len(m.containers) > 0 {
				ctr := m.containers[m.cursor]
				if ctr.State != "running" {
					go func() {
						ctx := context.Background()
						_ = m.collector.client.ContainerStart(ctx, ctr.ID, container.StartOptions{})
					}()
					m.loading = true
					return m, m.loadContainers
				}
			}

		case "x":
			// Stop selected container
			if len(m.containers) > 0 {
				ctr := m.containers[m.cursor]
				if ctr.State == "running" {
					go func() {
						ctx := context.Background()
						_ = m.collector.client.ContainerStop(ctx, ctr.ID, container.StopOptions{})
					}()
					m.loading = true
					return m, m.loadContainers
				}
			}

		case "enter":
			// Shell into container (exit TUI)
			if len(m.containers) > 0 && m.containers[m.cursor].State == "running" {
				m.cancel()
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		return m, tea.Batch(
			m.loadContainers,
			m.loadMetrics,
			m.tickCmd(),
		)

	case containerListMsg:
		m.containers = msg
		m.loading = false
		m.lastRefresh = time.Now()
		// Ensure cursor is valid
		if m.cursor >= len(m.containers) {
			m.cursor = max(0, len(m.containers)-1)
		}

	case metricsUpdateMsg:
		m.metrics = msg

	case eventMsg:
		m.events = append(m.events, msg)
		if len(m.events) > 10 {
			m.events = m.events[1:]
		}

	case errMsg:
		m.err = msg.err
		m.loading = false

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

// View renders the dashboard
func (m DashboardModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var sections []string

	// Header
	header := m.renderHeader()
	sections = append(sections, header)

	// Container list
	containerList := m.renderContainerList()
	sections = append(sections, containerList)

	// Metrics panel (for selected container)
	if len(m.containers) > 0 && m.cursor < len(m.containers) {
		metricsPanel := m.renderMetrics()
		sections = append(sections, metricsPanel)
	}

	// Help bar
	help := m.renderHelp()
	sections = append(sections, help)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m *DashboardModel) renderHeader() string {
	title := titleStyle.Render("📊 Container-Maker Monitor")

	status := ""
	if m.loading {
		status = m.spinner.View() + " Refreshing..."
	} else {
		status = mutedStyle.Render(fmt.Sprintf("Updated %s ago",
			time.Since(m.lastRefresh).Truncate(time.Second)))
	}

	running := 0
	for _, c := range m.containers {
		if c.State == "running" {
			running++
		}
	}

	summary := fmt.Sprintf("Containers: %d running / %d total", running, len(m.containers))
	summary = headerStyle.Render(summary)

	return lipgloss.JoinHorizontal(lipgloss.Center, title, "  ", summary, "  ", status)
}

func (m *DashboardModel) renderContainerList() string {
	if len(m.containers) == 0 {
		return borderStyle.Render("No containers found. Run 'cm env create' to create one.")
	}

	// Sort containers: running first, then by name
	sort.Slice(m.containers, func(i, j int) bool {
		if m.containers[i].State == "running" && m.containers[j].State != "running" {
			return true
		}
		if m.containers[i].State != "running" && m.containers[j].State == "running" {
			return false
		}
		return m.containers[i].Name < m.containers[j].Name
	})

	var rows []string

	// Header
	header := fmt.Sprintf("  %-3s %-20s %-10s %-8s %-8s %-12s %-20s",
		"", "NAME", "STATUS", "CPU", "MEM", "NET I/O", "IMAGE")
	rows = append(rows, headerStyle.Render(header))

	for i, c := range m.containers {
		// Status indicator
		statusIcon := "○"
		statusStyle := stoppedStyle
		if c.State == "running" {
			statusIcon = "●"
			statusStyle = runningStyle
		}
		statusIcon = statusStyle.Render(statusIcon)

		// Metrics
		cpu := "-"
		mem := "-"
		netIO := "-"
		if metrics, ok := m.metrics[c.ID]; ok {
			cpu = fmt.Sprintf("%.1f%%", metrics.CPUPercent)
			mem = formatBytes(metrics.MemoryUsed)
			netIO = fmt.Sprintf("%s/%s", formatBytes(metrics.NetworkRx), formatBytes(metrics.NetworkTx))
		}

		// Truncate image name
		image := c.Image
		if len(image) > 20 {
			image = image[:17] + "..."
		}

		// Build row
		row := fmt.Sprintf("%s %-20s %-10s %-8s %-8s %-12s %-20s",
			statusIcon,
			truncate(c.Name, 20),
			c.State,
			cpu,
			mem,
			netIO,
			image)

		if i == m.cursor {
			row = selectedStyle.Render(row)
		} else {
			row = normalStyle.Render(row)
		}

		rows = append(rows, row)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return borderStyle.Width(m.width - 4).Render(content)
}

func (m *DashboardModel) renderMetrics() string {
	if m.cursor >= len(m.containers) {
		return ""
	}

	c := m.containers[m.cursor]
	metrics, ok := m.metrics[c.ID]
	if !ok {
		return borderStyle.Render(fmt.Sprintf("No metrics available for %s", c.Name))
	}

	// CPU bar
	cpuBar := renderBar(metrics.CPUPercent, 100, 20)
	cpuLine := fmt.Sprintf("CPU:    %s %.1f%%", cpuBar, metrics.CPUPercent)

	// Memory bar
	memPercent := 0.0
	if metrics.MemoryLimit > 0 {
		memPercent = float64(metrics.MemoryUsed) / float64(metrics.MemoryLimit) * 100
	}
	memBar := renderBar(memPercent, 100, 20)
	memLine := fmt.Sprintf("Memory: %s %s / %s (%.1f%%)",
		memBar,
		formatBytes(metrics.MemoryUsed),
		formatBytes(metrics.MemoryLimit),
		memPercent)

	// Network I/O
	netLine := fmt.Sprintf("Network: ↓ %s ↑ %s", formatBytes(metrics.NetworkRx), formatBytes(metrics.NetworkTx))

	// Block I/O
	blockLine := fmt.Sprintf("Block:   R: %s W: %s", formatBytes(metrics.BlockRead), formatBytes(metrics.BlockWrite))

	// PIDs
	pidsLine := fmt.Sprintf("PIDs:    %d", metrics.PIDs)

	content := lipgloss.JoinVertical(lipgloss.Left,
		headerStyle.Render(" 📈 "+c.Name),
		"",
		cpuLine,
		memLine,
		netLine,
		blockLine,
		pidsLine,
	)

	return borderStyle.Width(m.width - 4).Render(content)
}

func (m *DashboardModel) renderHelp() string {
	commands := []string{
		"↑/↓ Navigate",
		"r Refresh",
		"s Start",
		"x Stop",
		"l Logs",
		"Enter Shell",
		"q Quit",
	}
	return helpStyle.Render(strings.Join(commands, " │ "))
}

// Command functions
func (m *DashboardModel) tickCmd() tea.Cmd {
	return tea.Tick(m.refreshRate, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *DashboardModel) loadContainers() tea.Msg {
	containers, err := m.collector.ListContainers(m.ctx, true)
	if err != nil {
		return errMsg{err}
	}
	// Filter to only show CM-managed containers
	var filtered []*ContainerInfo
	for _, c := range containers {
		if strings.HasPrefix(c.Name, "cm-") || c.Labels["cm.managed_by"] == "container-maker" {
			filtered = append(filtered, c)
		}
	}
	if len(filtered) == 0 {
		filtered = containers // Show all if no CM containers
	}
	return containerListMsg(filtered)
}

func (m *DashboardModel) loadMetrics() tea.Msg {
	metrics, err := m.collector.CollectAll(m.ctx)
	if err != nil {
		return errMsg{err}
	}
	result := make(map[string]*ContainerMetrics)
	for _, m := range metrics {
		result[m.ContainerID] = m
	}
	return metricsUpdateMsg(result)
}

func (m *DashboardModel) startLogs(_ string) tea.Cmd {
	return func() tea.Msg {
		// Log streaming would be implemented here
		return nil
	}
}

// Helper functions
func renderBar(value, max float64, width int) string {
	if max <= 0 {
		max = 100
	}
	ratio := value / max
	if ratio > 1 {
		ratio = 1
	}
	filled := int(ratio * float64(width))

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)

	// Color based on usage
	if ratio > 0.9 {
		return lipgloss.NewStyle().Foreground(colorDanger).Render(bar)
	} else if ratio > 0.7 {
		return lipgloss.NewStyle().Foreground(colorWarning).Render(bar)
	}
	return lipgloss.NewStyle().Foreground(colorSuccess).Render(bar)
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// RunDashboard starts the monitoring dashboard
func RunDashboard() error {
	model, err := NewDashboardModel()
	if err != nil {
		return err
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
