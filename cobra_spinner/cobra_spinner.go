// CobraSpinner provides a unified spinner to displaying while waiting for async
// operations. Do not use in a script context.
//
// Call New() to get a program, p.Run() to allow the program to take over the terminal (after
// spinning up the reaper goroutine), and p.Quit() from the reaper when done waiting.
package cobraspinner

import (
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
	return cs.spnr.View()
}

// Create a new BubbleTea program with just a spinner
//
// When you are done waiting, call p.Quit() from another goroutine.
func New() (p *tea.Program) {
	return tea.NewProgram(CobraSpinner{
		spnr: spinner.New(
			spinner.WithSpinner(spinner.Moon),
			spinner.WithStyle(lipgloss.NewStyle().Foreground(stylesheet.PrimaryColor)))})
}
