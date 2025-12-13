package tui

import (
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// StatusModel represents the status dashboard model
type StatusModel struct {
	containers []ContainerInfo
	selected   int
	width      int
	height     int
	quitting   bool
	loading    bool
	err        error
}

// ContainerInfo holds container display information
type ContainerInfo struct {
	ID      string
	Name    string
	Image   string
	Status  string
	Ports   string
	Created string
}

// NewStatusModel creates a new status dashboard model
func NewStatusModel() StatusModel {
	return StatusModel{
		loading: true,
	}
}

type containersLoadedMsg []ContainerInfo
type errMsg error

func loadContainers() tea.Msg {
	cmd := exec.Command("docker", "ps", "--format", "{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}\t{{.CreatedAt}}")
	output, err := cmd.Output()
	if err != nil {
		return errMsg(err)
	}

	var containers []ContainerInfo
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) >= 6 {
			containers = append(containers, ContainerInfo{
				ID:      parts[0],
				Name:    parts[1],
				Image:   parts[2],
				Status:  parts[3],
				Ports:   parts[4],
				Created: parts[5],
			})
		}
	}

	return containersLoadedMsg(containers)
}

func (m StatusModel) Init() tea.Cmd {
	return loadContainers
}

func (m StatusModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.containers)-1 {
				m.selected++
			}
		case "r":
			m.loading = true
			return m, loadContainers
		case "l":
			// View logs for selected container
			if len(m.containers) > 0 && m.selected < len(m.containers) {
				containerID := m.containers[m.selected].ID
				return m, tea.ExecProcess(exec.Command("docker", "logs", "-f", containerID), func(err error) tea.Msg {
					return nil
				})
			}
		case "s":
			// Shell into selected container
			if len(m.containers) > 0 && m.selected < len(m.containers) {
				containerID := m.containers[m.selected].ID
				return m, tea.ExecProcess(exec.Command("docker", "exec", "-it", containerID, "/bin/sh"), func(err error) tea.Msg {
					return nil
				})
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case containersLoadedMsg:
		m.loading = false
		m.containers = msg
	case errMsg:
		m.loading = false
		m.err = msg
	}
	return m, nil
}

func (m StatusModel) View() string {
	if m.quitting {
		return ""
	}

	var s strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(ColorPrimary).
		Padding(0, 2).
		Width(m.width)

	s.WriteString(headerStyle.Render("📦 Container-Make Status Dashboard"))
	s.WriteString("\n\n")

	if m.loading {
		s.WriteString(StyleInfo.Render("Loading containers..."))
		return s.String()
	}

	if m.err != nil {
		s.WriteString(StyleError.Render(fmt.Sprintf("Error: %v", m.err)))
		return s.String()
	}

	if len(m.containers) == 0 {
		s.WriteString(StyleSubtle.Render("No running containers found.\n"))
		s.WriteString(StyleSubtle.Render("Run 'cm run -- <command>' to start a container."))
		return s.String()
	}

	// Container list
	for i, c := range m.containers {
		cursor := "  "
		style := StyleSubtle

		if i == m.selected {
			cursor = "❯ "
			style = lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true)
		}

		// Format container info
		name := c.Name
		if len(name) > 20 {
			name = name[:17] + "..."
		}
		image := c.Image
		if len(image) > 30 {
			image = image[:27] + "..."
		}

		line := fmt.Sprintf("%s%-20s  %-30s  %s", cursor, name, image, c.Status)
		s.WriteString(style.Render(line))
		s.WriteString("\n")

		if i == m.selected {
			// Show details for selected container
			detailStyle := lipgloss.NewStyle().Foreground(ColorSubtle).PaddingLeft(4)
			s.WriteString(detailStyle.Render(fmt.Sprintf("ID: %s", c.ID)))
			s.WriteString("\n")
			if c.Ports != "" {
				s.WriteString(detailStyle.Render(fmt.Sprintf("Ports: %s", c.Ports)))
				s.WriteString("\n")
			}
			s.WriteString("\n")
		}
	}

	// Help
	s.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(ColorSubtle)
	s.WriteString(helpStyle.Render("↑/↓: Navigate  r: Refresh  l: Logs  s: Shell  q: Quit"))

	return s.String()
}

// RunStatusDashboard runs the status dashboard
func RunStatusDashboard() error {
	p := tea.NewProgram(NewStatusModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
