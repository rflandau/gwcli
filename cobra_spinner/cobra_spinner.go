package cobraspinner

import (
	"fmt"
	"gwcli/stylesheet"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type CobraSpinner struct {
	spnr spinner.Model
}

func (cs CobraSpinner) Init() tea.Cmd {
	return cs.spnr.Tick
}

func (cs CobraSpinner) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var toRet tea.Cmd
	cs.spnr, toRet = cs.spnr.Update(msg)
	return cs, toRet
}

func (cs CobraSpinner) View() string {
	return fmt.Sprintf("%s", cs.spnr.View())
}

// Generate a simple bubble tea model for the wait spinner
func New() CobraSpinner {
	s := spinner.New(
		spinner.WithSpinner(spinner.Moon),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(stylesheet.PrimaryColor)))

	return CobraSpinner{spnr: s}
}
