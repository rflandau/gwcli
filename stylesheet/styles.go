package stylesheet

import "github.com/charmbracelet/lipgloss"

var (
	NavStyle    = lipgloss.NewStyle().Foreground(NavColor)
	ActionStyle = lipgloss.NewStyle().Foreground(ActionColor)
	ErrStyle    = lipgloss.NewStyle().Foreground(ErrorColor)
	ModelStyle  = lipgloss.NewStyle(). // base style for a child model
			Align(lipgloss.Center, lipgloss.Center).BorderStyle(lipgloss.HiddenBorder())
	Composable = struct { // styles for multiple, simultaneous models
		Unfocused lipgloss.Style
		Focused   lipgloss.Style
	}{
		Unfocused: ModelStyle,
		Focused: ModelStyle.
			BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("69")),
	}
)
