package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RenderWelcome renders the welcome header
func RenderWelcome() {
	logo := `
   ______            __        _                      __  ___      __
  / ____/___  ____  / /_____ _(_)___  ___  _____     /  |/  /___ _/ /_____ 
 / /   / __ \/ __ \/ __/ __ ` + "`" + `/ / __ \/ _ \/ ___/____/ /|_/ / __ ` + "`" + `/ //_/ _ \
/ /___/ /_/ / / / / /_/ /_/ / / / / /  __/ /  /_____/ /  / / /_/ / ,< /  __/
\____/\____/_/ /_/\__/\__,_/_/_/ /_/\___/_/        /_/  /_/\__,_/_/|_|\___/ 
`

	// Gradient effect for logo (simulated with 2 colors)
	lines := strings.Split(logo, "\n")
	var logoOutput strings.Builder
	for i, line := range lines {
		if i%2 == 0 {
			logoOutput.WriteString(lipgloss.NewStyle().Foreground(ColorPrimary).Render(line))
		} else {
			logoOutput.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#9F86F6")).Render(line))
		}
		logoOutput.WriteString("\n")
	}

	fmt.Println(logoOutput.String())

	tagline := "🚀 The native DevContainer experience for Makefiles"
	fmt.Println(lipgloss.NewStyle().
		Foreground(ColorSubtle).
		Italic(true).
		Render(tagline))
	fmt.Println()
}

// RenderBox renders a content box with title
func RenderBox(title string, content string) {
	fmt.Println(StyleBox.Render(
		StyleTitle.Render(title) + "\n" + content,
	))
}
