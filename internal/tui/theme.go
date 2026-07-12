package tui

import "github.com/charmbracelet/lipgloss"

// ThemeStyles holds Lip Gloss styles for one color palette.
type ThemeStyles struct {
	Title  lipgloss.Style
	Subtle lipgloss.Style
	Panel  lipgloss.Style
	Active lipgloss.Style
	Error  lipgloss.Style
	Label  lipgloss.Style
}

var themeNames = []string{"midnight", "solar", "forest"}

// NormalizeTheme returns a supported theme name, defaulting to midnight.
func NormalizeTheme(name string) string {
	for _, candidate := range themeNames {
		if candidate == name {
			return candidate
		}
	}
	return "midnight"
}

// NextTheme cycles forward through supported themes.
func NextTheme(current string) string {
	current = NormalizeTheme(current)
	for i, name := range themeNames {
		if name == current {
			return themeNames[(i+1)%len(themeNames)]
		}
	}
	return themeNames[0]
}

// PrevTheme cycles backward through supported themes.
func PrevTheme(current string) string {
	current = NormalizeTheme(current)
	for i, name := range themeNames {
		if name == current {
			return themeNames[(i+len(themeNames)-1)%len(themeNames)]
		}
	}
	return themeNames[0]
}

// BuildThemeStyles returns Lip Gloss styles for the given theme name.
func BuildThemeStyles(name string) ThemeStyles {
	switch NormalizeTheme(name) {
	case "solar":
		return ThemeStyles{
			Title:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("236")).Background(lipgloss.Color("220")).Padding(0, 1),
			Subtle: lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
			Panel:  lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("214")).Padding(1, 2),
			Active: lipgloss.NewStyle().Foreground(lipgloss.Color("172")).Bold(true),
			Error:  lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
			Label:  lipgloss.NewStyle().Foreground(lipgloss.Color("235")).Bold(true),
		}
	case "forest":
		return ThemeStyles{
			Title:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("28")).Padding(0, 1),
			Subtle: lipgloss.NewStyle().Foreground(lipgloss.Color("242")),
			Panel:  lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("34")).Padding(1, 2),
			Active: lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true),
			Error:  lipgloss.NewStyle().Foreground(lipgloss.Color("203")),
			Label:  lipgloss.NewStyle().Foreground(lipgloss.Color("194")).Bold(true),
		}
	default:
		return ThemeStyles{
			Title:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62")).Padding(0, 1),
			Subtle: lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
			Panel:  lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62")).Padding(1, 2),
			Active: lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true),
			Error:  lipgloss.NewStyle().Foreground(lipgloss.Color("203")),
			Label:  lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Bold(true),
		}
	}
}
