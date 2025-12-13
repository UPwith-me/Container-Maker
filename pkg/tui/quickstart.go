package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/container-make/cm/pkg/template"
)

// QuickStartAction is a simpler menu item for the quickstart wizard
type QuickStartAction struct {
	Title       string
	Description string
	Key         string
}

type QuickStartModel struct {
	cursor   int
	actions  []QuickStartAction
	selected int
	quitting bool
}

func InitialQuickStartModel() QuickStartModel {
	return QuickStartModel{
		cursor: 0,
		actions: []QuickStartAction{
			{"python-basic", "Python 3.11 with pip", "1"},
			{"pytorch", "PyTorch with GPU support", "2"},
			{"node-basic", "Node.js 20 with npm", "3"},
			{"go-basic", "Go 1.21 development", "4"},
			{"Browse all templates", "Search 17+ templates", "5"},
			{"Custom image", "Use your own Docker image", "6"},
		},
	}
}

func (m QuickStartModel) Init() tea.Cmd {
	return nil
}

func (m QuickStartModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if m.cursor < len(m.actions)-1 {
				m.cursor++
			}
		case "enter", " ":
			m.selected = m.cursor + 1
			return m, tea.Quit
		case "1", "2", "3", "4", "5", "6":
			m.selected = int(msg.String()[0] - '0')
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m QuickStartModel) View() string {
	if m.selected != 0 {
		return ""
	}
	if m.quitting {
		return ""
	}

	var s strings.Builder

	// Styles
	warnBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FFC857")). // Yellow
		Padding(0, 1).
		MarginBottom(1)

	warnTitle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFC857")).Bold(true)

	// Header
	s.WriteString(warnBox.Render(warnTitle.Render("⚠️  No devcontainer.json found")))
	s.WriteString("\n")
	s.WriteString("Let's set one up! Choose an option:\n\n")

	// Menu
	for i, action := range m.actions {
		cursor := "  "
		itemStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B6B6B"))

		if m.cursor == i {
			cursor = "❯ "
			itemStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
		}

		title := itemStyle.Render(action.Title)
		desc := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Italic(true).Render(action.Description)

		s.WriteString(fmt.Sprintf("%s%s  %s\n", cursor, title, desc))
	}

	s.WriteString("\n")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Render("Use arrow keys to navigate • enter to select"))
	s.WriteString("\n")

	return s.String()
}

// RunQuickStart shows the quick start wizard
func RunQuickStart() error {
	p := tea.NewProgram(InitialQuickStartModel())
	m, err := p.Run()
	if err != nil {
		return err
	}

	model, ok := m.(QuickStartModel)
	if !ok || model.selected == 0 {
		return fmt.Errorf("setup cancelled")
	}

	// Handle selection
	switch model.selected {
	case 1:
		return applyQuickTemplate("python-basic")
	case 2:
		return applyQuickTemplate("pytorch")
	case 3:
		return applyQuickTemplate("node-basic")
	case 4:
		return applyQuickTemplate("go-basic")
	case 5:
		return browseTemplates()
	case 6:
		return customImage()
	}

	return nil
}

// Re-implement helper functions (applyQuickTemplate, browseTemplates, customImage)
// to ensure they are available in this package (previously they were in the same file)

func applyQuickTemplate(name string) error {
	cwd, _ := os.Getwd()
	fmt.Printf("📦 Applying template '%s'...\n", name)
	if err := template.ApplyTemplate(name, cwd); err != nil {
		return fmt.Errorf("failed to apply template: %w", err)
	}
	fmt.Println("✅ Created .devcontainer/devcontainer.json")
	return nil
}

func browseTemplates() error {
	// Simple browse for now, could be a TUI list later
	fmt.Println(template.ListTemplates())
	fmt.Print("\nEnter template name: ")
	var name string
	fmt.Scanln(&name)
	if name == "" {
		return nil
	}
	return applyQuickTemplate(name)
}

func customImage() error {
	fmt.Print("Enter Docker image: ")
	var imageName string
	fmt.Scanln(&imageName)
	if imageName == "" {
		return nil
	}

	content := fmt.Sprintf(`{
  "name": "Custom Dev Container",
  "image": "%s"
}`, imageName)

	os.MkdirAll(".devcontainer", 0755)
	os.WriteFile(".devcontainer/devcontainer.json", []byte(content), 0644)
	fmt.Println("✅ Created .devcontainer/devcontainer.json")
	return nil
}

func ShouldShowQuickStart() bool {
	if _, err := os.Stat(".devcontainer/devcontainer.json"); err == nil {
		return false
	}
	if _, err := os.Stat("devcontainer.json"); err == nil {
		return false
	}
	return true
}
