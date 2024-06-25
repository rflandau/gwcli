package stylesheet

import "github.com/charmbracelet/lipgloss"

var (
	NavStyle    = lipgloss.NewStyle().Foreground(NavColor)
	ActionStyle = lipgloss.NewStyle().Foreground(ActionColor)
	ErrStyle    = lipgloss.NewStyle().Foreground(ErrorColor)
	ModelStyle  = lipgloss.NewStyle(). // base box style for a child model
			Align(lipgloss.Left, lipgloss.Center).BorderStyle(lipgloss.HiddenBorder())
	Composable = struct { // styles for multiple, simultaneous models
		Unfocused lipgloss.Style
		Focused   lipgloss.Style
	}{
		Unfocused: ModelStyle,
		Focused: ModelStyle.
			BorderStyle(lipgloss.NormalBorder()).BorderForeground(AccentColor1),
	}
	Header1Style   = lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true)
	GreyedOutStyle = lipgloss.NewStyle().Faint(true)
	PromptStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color(PrimaryColor))
)
