package tui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/UPwith-me/Container-Maker/pkg/team"
	"github.com/UPwith-me/Container-Maker/pkg/template"
	"github.com/UPwith-me/Container-Maker/pkg/userconfig"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TemplateChoice represents a template option in the wizard
type TemplateChoice struct {
	Name        string
	Description string
	Category    string // "team/reponame", "Official", etc.
	TemplateID  string // For applying template
	IsTeam      bool
}

type InitModel struct {
	cursor   int
	choices  []TemplateChoice
	selected *TemplateChoice
	quitting bool
	orgName  string
}

// Styles
var (
	categoryStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			MarginTop(1)

	teamCategoryStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#00CED1")).
				MarginTop(1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575")).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B6B6B"))

	descStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))
)

func InitialInitModel() InitModel {
	var choices []TemplateChoice
	var orgName string

	// Load user config for org name
	if cfg, err := userconfig.Load(); err == nil {
		orgName = cfg.Team.OrgName
	}

	// 1. Load team templates first (highest priority)
	teamTemplates, _ := team.GetAllTeamTemplates()

	// Sort repos by name for consistent ordering
	var repoNames []string
	for name := range teamTemplates {
		repoNames = append(repoNames, name)
	}
	sort.Strings(repoNames)

	for _, repoName := range repoNames {
		templates := teamTemplates[repoName]
		for _, t := range templates {
			desc := t.Description
			if desc == "" {
				desc = "Team template"
			}
			choices = append(choices, TemplateChoice{
				Name:        t.Name,
				Description: desc,
				Category:    fmt.Sprintf("team/%s", repoName),
				TemplateID:  fmt.Sprintf("team/%s/%s", repoName, t.Name),
				IsTeam:      true,
			})
		}
	}

	// 2. Load official templates
	officialTemplates := template.BuiltInTemplates()

	// Group by category
	categoryOrder := []string{"Deep Learning", "Python", "Go", "Node.js", "Rust", "C++", "Java", ".NET", "PHP", "Ruby"}
	categoryTemplates := make(map[string][]template.Template)

	for _, t := range officialTemplates {
		categoryTemplates[t.Category] = append(categoryTemplates[t.Category], *t)
	}

	for _, cat := range categoryOrder {
		templates := categoryTemplates[cat]
		if len(templates) == 0 {
			continue
		}

		// Sort by name
		sort.Slice(templates, func(i, j int) bool {
			return templates[i].Name < templates[j].Name
		})

		for _, t := range templates {
			choices = append(choices, TemplateChoice{
				Name:        t.Name,
				Description: t.Description,
				Category:    cat,
				TemplateID:  t.Name,
				IsTeam:      false,
			})
		}
	}

	return InitModel{
		choices: choices,
		orgName: orgName,
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
			if m.cursor < len(m.choices) {
				m.selected = &m.choices[m.cursor]
			}
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m InitModel) View() string {
	if m.selected != nil {
		return "" // Clear view on success
	}
	if m.quitting {
		return "Init cancelled.\n"
	}

	s := strings.Builder{}
	s.WriteString(StyleTitle.Render("Select a DevContainer Template:"))
	s.WriteString("\n")

	// Render choices grouped by category
	lastCategory := ""
	for i, choice := range m.choices {
		// Category header
		if choice.Category != lastCategory {
			lastCategory = choice.Category

			if choice.IsTeam {
				// Team category with special styling
				displayName := choice.Category
				if m.orgName != "" {
					displayName = fmt.Sprintf("[%s] %s", m.orgName, strings.TrimPrefix(choice.Category, "team/"))
				}
				s.WriteString(teamCategoryStyle.Render(fmt.Sprintf("\n  === %s ===", displayName)))
			} else {
				s.WriteString(categoryStyle.Render(fmt.Sprintf("\n  --- %s ---", choice.Category)))
			}
			s.WriteString("\n")
		}

		// Template entry
		cursor := "  "
		name := choice.Name
		desc := choice.Description

		if m.cursor == i {
			cursor = "> "
			name = selectedStyle.Render(name)
			desc = descStyle.Render(desc)
		} else {
			name = dimStyle.Render(name)
			desc = dimStyle.Render(desc)
		}

		// Truncate description if too long
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}

		s.WriteString(fmt.Sprintf("  %s%-20s %s\n", cursor, name, desc))
	}

	s.WriteString("\n")
	s.WriteString(dimStyle.Render("  [arrows] navigate  [enter] select  [q] quit"))
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

	if model, ok := m.(InitModel); ok && model.selected != nil {
		return model.selected.TemplateID, nil
	}

	return "", nil // Cancelled
}

// GetSelectedTemplate returns full template info
func RunInitWizardWithDetails() (*TemplateChoice, error) {
	p := tea.NewProgram(InitialInitModel())
	m, err := p.Run()
	if err != nil {
		return nil, err
	}

	if model, ok := m.(InitModel); ok && model.selected != nil {
		return model.selected, nil
	}

	return nil, nil
}

// GenerateConfig generates the devcontainer.json content based on selection
func GenerateConfig(templateID string) string {
	// Check if it's a team template
	if strings.HasPrefix(templateID, "team/") {
		parts := strings.SplitN(strings.TrimPrefix(templateID, "team/"), "/", 2)
		if len(parts) == 2 {
			repoName, templateName := parts[0], parts[1]
			if templatePath, err := team.GetTemplatePath(repoName, templateName); err == nil {
				// Return path instead - caller should copy files
				return fmt.Sprintf(`{"_teamTemplatePath": "%s"}`, templatePath)
			}
		}
	}

	// Official template
	if t, ok := template.BuiltInTemplates()[templateID]; ok {
		return generateFromTemplate(t)
	}

	// Fallback
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
	case strings.Contains(templateID, "go"):
		return fmt.Sprintf(base, "Go Project", "mcr.microsoft.com/devcontainers/go:1.21")
	case strings.Contains(templateID, "python"):
		return fmt.Sprintf(base, "Python Project", "mcr.microsoft.com/devcontainers/python:3.11")
	case strings.Contains(templateID, "node"):
		return fmt.Sprintf(base, "Node.js Project", "mcr.microsoft.com/devcontainers/javascript-node:18")
	case strings.Contains(templateID, "rust"):
		return fmt.Sprintf(base, "Rust Project", "mcr.microsoft.com/devcontainers/rust:latest")
	case strings.Contains(templateID, "cpp"):
		return fmt.Sprintf(base, "C++ Project", "mcr.microsoft.com/devcontainers/cpp:ubuntu-22.04")
	default:
		return fmt.Sprintf(base, "Dev Container", "mcr.microsoft.com/devcontainers/base:debian")
	}
}

// generateFromTemplate creates devcontainer.json from a template struct
func generateFromTemplate(t *template.Template) string {
	config := map[string]interface{}{
		"name":  t.Name,
		"image": t.Image,
	}

	if len(t.RunArgs) > 0 {
		config["runArgs"] = t.RunArgs
	}

	if t.PostCreate != "" {
		config["postCreateCommand"] = t.PostCreate
	}

	if len(t.Features) > 0 {
		config["features"] = t.Features
	}

	// Add VS Code customizations
	config["customizations"] = map[string]interface{}{
		"vscode": map[string]interface{}{
			"extensions": []string{},
		},
	}

	data, _ := json.MarshalIndent(config, "", "  ")
	return string(data)
}
