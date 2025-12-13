package tui

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
	ColorPrimary   = lipgloss.Color("#7D56F4") // Purple
	ColorSecondary = lipgloss.Color("#04B575") // Green
	ColorError     = lipgloss.Color("#FF4672") // Red
	ColorWarning   = lipgloss.Color("#FFC857") // Yellow
	ColorSubtle    = lipgloss.Color("#6B6B6B") // Gray
)

// Common styles
var (
	// Output styles
	StyleSuccess = lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true)
	StyleError   = lipgloss.NewStyle().Foreground(ColorError).Bold(true)
	StyleWarning = lipgloss.NewStyle().Foreground(ColorWarning)
	StyleInfo    = lipgloss.NewStyle().Foreground(ColorPrimary)
	StyleSubtle  = lipgloss.NewStyle().Foreground(ColorSubtle)

	// Box styles
	StyleBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2)

	// Header styles
	StyleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(ColorPrimary).
			Padding(0, 1)

	StyleTitle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			MarginBottom(1)
)

// Icons
const (
	IconSuccess = "✅"
	IconError   = "❌"
	IconWarning = "⚠️"
	IconInfo    = "ℹ️"
	IconBox     = "📦"
	IconRocket  = "🚀"
)
