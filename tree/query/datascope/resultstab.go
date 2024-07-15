package datascope

/**
 * Subroutines for driving and displaying the results tab.
 */

import (
	"fmt"
	"gwcli/stylesheet"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func updateResults(s *DataScope, msg tea.Msg) tea.Cmd {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)
	prevPage := s.pager.Page
	s.pager, cmd = s.pager.Update(msg)
	cmds = append(cmds, cmd)

	s.setResultsDisplayed() // pass the new content to the view

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

// Determines and sets the the content currently visible in the results viewport.
func (s *DataScope) setResultsDisplayed() {
	start, end := s.pager.GetSliceBounds(len(s.data))
	data := s.data[start:end]

	// apply alterating color scheme
	var bldr strings.Builder
	var trueIndex int = start // index of full results, between start and end
	for _, d := range data {
		bldr.WriteString(indexStyle.Render(strconv.Itoa(trueIndex+1) + ":"))
		if trueIndex%2 == 0 {
			bldr.WriteString(evenEntryStyle.Render(d))
		} else {
			bldr.WriteString(oddEntryStyle.Render(d))
		}
		bldr.WriteRune('\n')
		trueIndex += 1
	}
	s.vp.SetContent(wrap(s.vp.Width, bldr.String()))
}

var compiledShortHelp = stylesheet.GreyedOutStyle.Render(
	fmt.Sprintf("%v page • %v scroll • tab: cycle • esc: quit",
		stylesheet.LeftRight, stylesheet.UpDown),
)

// generates a renderFooter with the box+line and help keys
func (s *DataScope) renderFooter(width int) string {
	var alignerSty = lipgloss.NewStyle().Width(s.vp.Width).AlignHorizontal(lipgloss.Center)
	// set up each element
	pageNumber := lipgloss.NewStyle().Foreground(stylesheet.FocusedColor).
		Render(strconv.Itoa(s.pager.Page)) + " "
	scrollPercent := fmt.Sprintf("%3.f%%", s.vp.ScrollPercent()*100)
	line := lipgloss.NewStyle().
		Foreground(stylesheet.PrimaryColor).
		Render(
			strings.Repeat("─",
				max(0, width-
					lipgloss.Width(pageNumber)-
					lipgloss.Width(scrollPercent))),
		)

	composedLine := fmt.Sprintf("%s%s%s", pageNumber, line, scrollPercent)

	return lipgloss.JoinVertical(lipgloss.Center,
		composedLine,
		alignerSty.Render(s.pager.View()),
		alignerSty.Render(compiledShortHelp),
	)
}
