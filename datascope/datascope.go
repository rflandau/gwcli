// Datascope is a combination of the scrolling viewport and paginator.
// It displays arbitrary data, one page at a time, in the alt buffer.
// As the user pages through, the viewport automatically updates with the contents of the new page.
//
// Like busywait, this can be invoked for Cobra or for Mother.
//
// TODO Add support and keys for downloading all data or just the current page of data
package datascope

import (
	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

//#region For Cobra Usage

type DataScope struct {
	vp    viewport.Model
	pager paginator.Model
	ready bool
	data  []string
}

func (s DataScope) Init() tea.Cmd {
	return nil
}

func (s DataScope) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return s, nil
}

func (s DataScope) View() string {
	return ""
}

func CobraNew(data []string) (p *tea.Program) {
	return tea.NewProgram(NewDataScope(data))
}

//#endregion For Cobra Usage

func NewDataScope(data []string) DataScope {
	// set up backend paginator
	p := paginator.New()
	p.Type = paginator.Dots
	p.PerPage = 30
	p.ActiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "235", Dark: "252"}).Render("•")
	p.InactiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "250", Dark: "238"}).Render("•")
	p.SetTotalPages(len(data))

	return DataScope{pager: p}
}
