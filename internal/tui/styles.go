package tui

import "github.com/charmbracelet/lipgloss"

// Adaptive shades keep the interface readable on light and dark terminals.
var (
	teal         = lipgloss.AdaptiveColor{Light: "30", Dark: "43"}
	amber        = lipgloss.AdaptiveColor{Light: "136", Dark: "214"}
	errorRed     = lipgloss.AdaptiveColor{Light: "160", Dark: "203"}
	headingStyle = lipgloss.NewStyle().Bold(true).Foreground(teal)
	warningStyle = lipgloss.NewStyle().Bold(true).Foreground(amber)
	errorStyle   = lipgloss.NewStyle().Foreground(errorRed)
)

func environmentLabel(env string) string {
	if env == "production" {
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("0")).Background(lipgloss.Color("214")).Padding(0, 1).Render("PRODUCTION")
	}
	return headingStyle.Render("SANDBOX")
}
