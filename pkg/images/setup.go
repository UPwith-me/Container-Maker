package images

import (
	"context"
	"fmt"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

// SetupModel is the Bubble Tea model for the setup wizard
type SetupModel struct {
	presets  []string
	selected map[string]bool
	cursor   int
	done     bool
	quitting bool
}

// NewSetupModel creates a new setup wizard model
func NewSetupModel() SetupModel {
	presets := []string{"go", "python", "node", "rust", "cpp", "base", "devcontainer"}
	selected := make(map[string]bool)

	// Pre-select common ones
	selected["go"] = true
	selected["python"] = true
	selected["base"] = true

	return SetupModel{
		presets:  presets,
		selected: selected,
	}
}

func (m SetupModel) Init() tea.Cmd {
	return nil
}

func (m SetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if m.cursor < len(m.presets)-1 {
				m.cursor++
			}
		case " ":
			name := m.presets[m.cursor]
			m.selected[name] = !m.selected[name]
		case "enter":
			m.done = true
			return m, tea.Quit
		case "a":
			// Select all
			for _, name := range m.presets {
				m.selected[name] = true
			}
		case "n":
			// Select none
			for _, name := range m.presets {
				m.selected[name] = false
			}
		}
	}
	return m, nil
}

func (m SetupModel) View() string {
	if m.quitting {
		return ""
	}

	defaults := DefaultPresets()

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00D4AA"))
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00D4AA"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))

	var sb strings.Builder

	sb.WriteString(titleStyle.Render("🎯 Container-Make Setup Wizard"))
	sb.WriteString("\n\n")
	sb.WriteString("Select images to download (space=toggle, enter=confirm):\n\n")

	for i, name := range m.presets {
		preset := defaults[name]

		cursor := "  "
		if i == m.cursor {
			cursor = "❯ "
		}

		checkbox := "[ ]"
		lineStyle := dimStyle
		if m.selected[name] {
			checkbox = "[✓]"
			lineStyle = selectedStyle
		}

		line := fmt.Sprintf("%s%s %-12s %-30s %s",
			cursor, checkbox, name, preset.Image, preset.Size)

		sb.WriteString(lineStyle.Render(line))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(dimStyle.Render("a=select all  n=select none  q=quit"))

	return sb.String()
}

// GetSelectedImages returns the list of selected image names
func (m SetupModel) GetSelectedImages() []string {
	var selected []string
	for name, isSelected := range m.selected {
		if isSelected {
			selected = append(selected, name)
		}
	}
	return selected
}

// RunSetupWizard runs the interactive setup wizard
func RunSetupWizard() ([]string, error) {
	model := NewSetupModel()
	p := tea.NewProgram(model)
	result, err := p.Run()
	if err != nil {
		return nil, err
	}

	finalModel := result.(SetupModel)
	if finalModel.quitting {
		return nil, nil
	}

	return finalModel.GetSelectedImages(), nil
}

// PullImage pulls a Docker image with progress display
func PullImage(imageName string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()

	fmt.Printf("  📥 Pulling %s...\n", imageName)

	reader, err := cli.ImagePull(context.Background(), imageName, image.PullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()

	// Just consume the output (we already have our own progress indicator)
	io.Copy(io.Discard, reader)

	fmt.Printf("  ✅ %s downloaded\n", imageName)
	return nil
}

// PullSelectedImages pulls all selected images
func PullSelectedImages(names []string) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	defaults := DefaultPresets()

	fmt.Printf("\n📥 Downloading %d images...\n\n", len(names))

	for _, name := range names {
		preset, ok := defaults[name]
		if !ok {
			continue
		}

		if err := PullImage(preset.Image); err != nil {
			fmt.Printf("  ❌ Failed to pull %s: %v\n", name, err)
		} else {
			config.Presets[name].Downloaded = true
		}
	}

	// Set default if not set
	if config.Default == "" && len(names) > 0 {
		config.Default = names[0]
	}

	SaveConfig(config)

	fmt.Println("\n🎉 Setup complete!")
	fmt.Println("   Use 'cm images use <name>' to switch images")
	fmt.Println("   Use 'cm images' to see all available images")

	return nil
}
