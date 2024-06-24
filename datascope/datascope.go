// Datascope is a combination of the scrolling viewport and paginator.
// It displays arbitrary data, one page at a time, in the alt buffer.
// As the user pages through, the viewport automatically updates with the contents of the new page.
//
// Like busywait, this can be invoked for Cobra or for Mother.
//
// TODO Add support and keys for downloading all data or just the current page of data
package datascope

import (
	"fmt"
	"gwcli/stylesheet"
	"strings"

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
	done  bool
	Title string
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

// Generates a string representation of the top margin and header box
func (s DataScope) header() string {
	title := viewportHeaderBoxStyle.Render(s.Title)
	line := strings.Repeat("─", max(0, s.vp.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

// Generates a string representation of the bottom margin and footer box
func (s DataScope) footer() string {
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", s.vp.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, s.vp.Width-lipgloss.Width(info)))

	upper := s.pager.View() + "\n"
	lower := "h/l ←/→ page • q: quit\n"
	builtLine := lipgloss.JoinVertical(lipgloss.Center, upper, line, lower)

	return lipgloss.JoinHorizontal(lipgloss.Center, builtLine, info)
}

func (s DataScope) Done() bool {
	return s.done
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

// #region styling
var viewportHeaderBoxStyle = func() lipgloss.Style {
	b := lipgloss.RoundedBorder()
	b.Right = "├"
	return lipgloss.NewStyle().BorderStyle(b).
		Padding(0, 1).
		BorderForeground(stylesheet.PrimaryColor)
}()

var infoStyle = func() lipgloss.Style {
	b := lipgloss.RoundedBorder()
	b.Left = "┤"
	return viewportHeaderBoxStyle.BorderStyle(b)
}()

//#endregion
