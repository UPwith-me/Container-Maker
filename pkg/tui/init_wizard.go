package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type InitModel struct {
	cursor   int
	choices  []string
	selected string
	quitting bool
}

func InitialInitModel() InitModel {
	return InitModel{
		choices: []string{
			"Go (1.21)",
			"Python (3.11)",
			"Node.js (18)",
			"Rust (latest)",
			"C++ (Ubuntu 22.04 base)",
			"Empty (Just base image)",
		},
	}
}

func (m InitModel) Init() tea.Cmd {
	return nil
}

func (m InitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter", " ":
			m.selected = m.choices[m.cursor]
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m InitModel) View() string {
	if m.selected != "" {
		return "" // Clear view on success to print final message cleanly
	}
	if m.quitting {
		return "Init cancelled.\n"
	}

	s := strings.Builder{}
	s.WriteString(StyleTitle.Render("Select a DevContainer Template:"))
	s.WriteString("\n\n")

	for i, choice := range m.choices {
		cursor := " " // no cursor
		if m.cursor == i {
			cursor = "❯" // cursor
			choice = lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render(choice)
		} else {
			choice = StyleSubtle.Render(choice)
		}
		s.WriteString(fmt.Sprintf("%s %s\n", cursor, choice))
	}

	s.WriteString("\n")
	s.WriteString(StyleSubtle.Render("Use arrow keys to navigate, enter to select."))
	s.WriteString("\n")

	return s.String()
}

// RunInitWizard starts the interactive init wizard
func RunInitWizard() (string, error) {
	p := tea.NewProgram(InitialInitModel())
	m, err := p.Run()
	if err != nil {
		return "", err
	}

	if model, ok := m.(InitModel); ok && model.selected != "" {
		return model.selected, nil
	}

	return "", nil // Cancelled or error
}

// GenerateConfig generates the devcontainer.json content based on selection
func GenerateConfig(template string) string {
	base := `{
	"name": "%s",
	"image": "%s",
	"customizations": {
		"vscode": {
			"extensions": []
		}
	}
}`

	switch {
	case strings.Contains(template, "Go"):
		return fmt.Sprintf(base, "Go Project", "mcr.microsoft.com/devcontainers/go:1.21")
	case strings.Contains(template, "Python"):
		return fmt.Sprintf(base, "Python Project", "mcr.microsoft.com/devcontainers/python:3.11")
	case strings.Contains(template, "Node"):
		return fmt.Sprintf(base, "Node.js Project", "mcr.microsoft.com/devcontainers/javascript-node:18")
	case strings.Contains(template, "Rust"):
		return fmt.Sprintf(base, "Rust Project", "mcr.microsoft.com/devcontainers/rust:latest")
	case strings.Contains(template, "C++"):
		return fmt.Sprintf(base, "C++ Project", "mcr.microsoft.com/devcontainers/cpp:ubuntu-22.04")
	default:
		return fmt.Sprintf(base, "Debian Project", "mcr.microsoft.com/devcontainers/base:debian")
	}
}
