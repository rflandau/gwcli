package datascope

import (
	"fmt"
	"gwcli/stylesheet"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tab struct {
	name string

	// update and view must take a DS parameter, rather than being methods of DS, as DS's elm arch
	// passes by value
	updateFunc func(*DataScope, tea.Msg) tea.Cmd
	// see the note on updateFunc
	viewFunc func(*DataScope) string
}

const (
	results uint = iota
	help
	download
	schedule
)

// results the array of tabs with all requisite data built in
func (s *DataScope) generateTabs() []tab {
	t := make([]tab, 4)
	t[results] = tab{name: "results", updateFunc: updateResults, viewFunc: viewResults}
	t[help] = tab{
		name:       "help",
		updateFunc: func(*DataScope, tea.Msg) tea.Cmd { return nil },
		viewFunc: func(*DataScope) string {
			return fmt.Sprintf("q: return to results view\n"+
				"tab: cycle tabs\n"+
				"shift+tab: reverse cycle tab\n"+
				"%v page\n"+
				"%v scroll\n"+
				"t: toggle tabs\n"+
				"esc: quit",
				stylesheet.LeftRight, stylesheet.UpDown)
		}}
	t[download] = tab{
		name: "download",
		updateFunc: func(*DataScope, tea.Msg) tea.Cmd {
			return nil
		},
		viewFunc: func(*DataScope) string { return "" }}
	t[schedule] = tab{
		name:       "schedule",
		updateFunc: func(*DataScope, tea.Msg) tea.Cmd { return nil },
		viewFunc:   func(*DataScope) string { return "" }}

	return t
}

func updateResults(s *DataScope, msg tea.Msg) tea.Cmd {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)
	prevPage := s.pager.Page
	s.pager, cmd = s.pager.Update(msg)
	cmds = append(cmds, cmd)
	// pass the new content to the view
	s.vp.SetContent(s.displayPage())
	s.vp, cmd = s.vp.Update(msg)
	cmds = append(cmds, cmd)
	if prevPage != s.pager.Page { // if page changed, reset to top of view
		s.vp.GotoTop()
	}
	return tea.Sequence(cmds...)
}

// view when 'results' tab is active
func viewResults(s *DataScope) string {
	if !s.ready {
		return "\nInitializing..."
	}
	return fmt.Sprintf("%s\n%s", s.vp.View(), s.renderFooter(s.vp.Width))
}

// #region tab drawing
func tabBorderWithBottom(left, middle, right string) lipgloss.Border {
	border := lipgloss.RoundedBorder()
	border.BottomLeft = left
	border.Bottom = middle
	border.BottomRight = right
	return border
}

var (
	inactiveTabBorder = tabBorderWithBottom("┴", "─", "┴")
	activeTabBorder   = tabBorderWithBottom("┘", " ", "└")
	highlightColor    = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	inactiveTabStyle  = lipgloss.NewStyle().Border(inactiveTabBorder, true).
				BorderForeground(highlightColor).
				Padding(0, 1).AlignHorizontal(lipgloss.Center)
	activeTabStyle = inactiveTabStyle.Border(activeTabBorder, true)
)

func (s *DataScope) renderTabs(width int) string {

	var rendered []string = make([]string, len(s.tabs))

	margin, tabCount := 2, len(s.tabs)

	// width = (tab_width * tab_count) + (margin * tab_count-1)
	// tab_width = (width - margin*(tab_count-1))/tab_count
	tabWidth := (width - (margin*tabCount - 1)) / tabCount
	// iterate and draw each tab, with special styling on the active tab
	for i, t := range s.tabs {
		var style lipgloss.Style
		isFirst, isLast, isActive := i == 0, i == len(s.tabs)-1, i == int(s.activeTab)
		if isActive {
			style = activeTabStyle
		} else {
			style = inactiveTabStyle
		}
		style = style.Width(tabWidth)
		border, _, _, _, _ := style.GetBorder()
		if isFirst && isActive {
			border.BottomLeft = "╵"
		} else if isFirst && !isActive {
			border.BottomLeft = "└"
		} else if isLast && isActive {
			border.BottomRight = "╵"
		} else if isLast && !isActive {
			border.BottomRight = "┘"
		}
		style = style.Border(border)
		if i == int(results) {
			rendered[i] = style.Foreground(stylesheet.AccentColor1).Render(t.name)
		} else {
			rendered[i] = style.Render(t.name)
		}

	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
	return lipgloss.NewStyle().AlignHorizontal(lipgloss.Left).Render(row)
}

//#endregion
