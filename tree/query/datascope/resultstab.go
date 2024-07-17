package datascope

/**
 * Subroutines for driving and displaying the results tab.
 */

import (
	"fmt"
	"gwcli/stylesheet"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type resultsTab struct {
	vp    viewport.Model
	pager paginator.Model
	data  []string // complete set of data to be paged
	ready bool
}

func initResultsTab(data []string) resultsTab {
	// set up backend paginator
	paginator.DefaultKeyMap = paginator.KeyMap{ // do not use pgup/pgdn
		PrevPage: key.NewBinding(key.WithKeys("left", "h")),
		NextPage: key.NewBinding(key.WithKeys("right", "l")),
	}
	p := paginator.New()
	p.Type = paginator.Dots
	p.PerPage = 25
	p.ActiveDot = lipgloss.NewStyle().Foreground(stylesheet.FocusedColor).Render("•")
	p.InactiveDot = lipgloss.NewStyle().Foreground(stylesheet.UnfocusedColor).Render("•")
	p.SetTotalPages(len(data))

	// set up viewport
	vp := viewport.Model{
		// width/height are set later
		// when received in a windowSize message
	}
	vp.MouseWheelDelta = 1
	vp.HighPerformanceRendering = false
	// set up keybinds directly supported by viewport
	// other keybinds are managed by the results tab()
	vp.KeyMap = viewport.KeyMap{
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", " ", "f"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "b"),
		),
		HalfPageUp: key.NewBinding(
			key.WithKeys("u", "ctrl+u"),
		),
		HalfPageDown: key.NewBinding(
			key.WithKeys("d", "ctrl+d"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
		),
	}

	r := resultsTab{
		vp:    vp,
		pager: p,
		data:  data,
	}

	return r
}

func updateResults(s *DataScope, msg tea.Msg) tea.Cmd {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	// handle pager modifications first
	prevPage := s.results.pager.Page
	s.results.pager, cmd = s.results.pager.Update(msg)
	cmds = append(cmds, cmd)

	s.setResultsDisplayed()               // pass the new content to the view
	if prevPage != s.results.pager.Page { // if page changed, reset to top of view
		s.results.vp.GotoTop()
	}

	// check for keybinds not directly supported by the viewport
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.Type {
		case tea.KeyHome:
			s.results.vp.GotoTop()
			return cmds[0]
		case tea.KeyEnd:
			s.results.vp.GotoBottom()
			return cmds[0]
		}
	}

	s.results.vp, cmd = s.results.vp.Update(msg)
	cmds = append(cmds, cmd)
	return tea.Sequence(cmds...)
}

// view when 'results' tab is active
func viewResults(s *DataScope) string {
	if !s.results.ready {
		return "\nInitializing..."
	}
	return fmt.Sprintf("%s\n%s", s.results.vp.View(), s.renderFooter(s.results.vp.Width))
}

// Determines and sets the the content currently visible in the results viewport.
func (s *DataScope) setResultsDisplayed() {
	start, end := s.results.pager.GetSliceBounds(len(s.results.data))
	data := s.results.data[start:end]

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
	s.results.vp.SetContent(wrap(s.results.vp.Width, bldr.String()))
}

var compiledShortHelp = stylesheet.GreyedOutStyle.Render(
	fmt.Sprintf("%v page • %v scroll • Home: Jump Top • End: Jump Bottom\n"+
		"tab: cycle • esc: quit",
		stylesheet.LeftRight, stylesheet.UpDown),
)

// generates a renderFooter with the box+line and help keys
func (s *DataScope) renderFooter(width int) string {
	var alignerSty = lipgloss.NewStyle().Width(s.results.vp.Width).AlignHorizontal(lipgloss.Center)
	// set up each element
	pageNumber := lipgloss.NewStyle().Foreground(stylesheet.FocusedColor).
		Render(strconv.Itoa(s.results.pager.Page+1)) + " "
	scrollPercent := fmt.Sprintf("%3.f%%", s.results.vp.ScrollPercent()*100)
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
		alignerSty.Render(s.results.pager.View()),
		alignerSty.Render(compiledShortHelp),
	)
}

// recalculate the dimensions of the results tab, factoring in results-specific margins.
// The clipped height is the height available to the results tab (height - tabs height).
func (s *DataScope) recalculateSize(rawWidth, clippedHeight int) {
	s.results.vp.Height = clippedHeight - lipgloss.Height(s.renderFooter(rawWidth))
	s.results.vp.Width = rawWidth
	s.results.ready = true
}
