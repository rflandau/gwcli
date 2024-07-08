// Datascope is tabbed, scrolling viewport with a paginator built into the results view.
// It displays arbitrary data, one page at a time, in the alt buffer.
// As the user pages through, the viewport automatically updates with the contents of the new page.
// The first tab contains the actual results, while
//
// Like busywait, this can be invoked for Cobra or for Mother.
package datascope

import (
	"fmt"
	"gwcli/clilog"
	"gwcli/stylesheet"
	"gwcli/utilities/killer"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
)

type DataScope struct {
	vp            viewport.Model
	pager         paginator.Model
	ready         bool
	data          []string // complete set of data to be paged
	motherRunning bool     // without Mother's support, we need to handle killkeys and death alone
	ftrHeight     int
	marginHeight  int

	tabs      []tab
	showTabs  bool
	activeTab uint
}

func NewDataScope(data []string, motherRunning bool) (DataScope, tea.Cmd) {
	// set up backend paginator
	p := paginator.New()
	p.Type = paginator.Dots
	p.PerPage = 30
	p.ActiveDot = lipgloss.NewStyle().Foreground(stylesheet.FocusedColor).Render("•")
	p.InactiveDot = lipgloss.NewStyle().Foreground(stylesheet.UnfocusedColor).Render("•")
	p.SetTotalPages(len(data))

	s := DataScope{
		pager:         p,
		ready:         false,
		data:          data,
		motherRunning: motherRunning,
	}

	// set up tabs
	s.tabs = s.generateTabs()
	s.activeTab = results

	// pre-set heights
	s.ftrHeight = lipgloss.Height(s.footer())
	s.marginHeight = s.ftrHeight // extra space not showing content // TODO include tab height

	// mother does not start in alt screen, and thus requires manual measurements
	if motherRunning {
		return s, tea.Sequence(tea.EnterAltScreen, func() tea.Msg {
			w, h, err := term.GetSize(os.Stdin.Fd())
			if err != nil {
				clilog.Writer.Errorf("Failed to fetch terminal size: %v", err)
			}
			return tea.WindowSizeMsg{Width: w, Height: h}
		})
	}
	return s, nil

}

func (s DataScope) Init() tea.Cmd {
	return nil
}

func (s DataScope) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	//clilog.Writer.Debugf("msg: %v\ntab: %v", msg, s.tabs[s.activeTab].name)

	// mother takes care of kill keys if she is running
	if !s.motherRunning {
		if kill := killer.CheckKillKeys(msg); kill != killer.None {
			clilog.Writer.Infof("Self-handled kill key, with kill type %v", kill)
			return s, tea.Batch(tea.Quit, tea.ExitAltScreen)
		}
	}

	switch msg := msg.(type) {
	case tea.KeyMsg: // tab-agnostic keys
		if key.Matches(msg, showTabsKey) {
			s.showTabs = true
			return s, nil
		}
	case tea.WindowSizeMsg:
		if !s.ready { // if we are not ready, use these dimensions to become ready
			s.vp = viewport.New(msg.Width, msg.Height-s.marginHeight)
			s.vp.HighPerformanceRendering = false
			s.vp.SetContent(s.displayPage())
			s.ready = true
		} else { // just an update
			s.vp.Width = msg.Width
			s.vp.Height = msg.Height - s.marginHeight
		}
	}

	return s, s.tabs[s.activeTab].updateFunc(&s, msg)
}

func (s DataScope) View() string {
	return s.tabs[s.activeTab].viewFunc(&s)
}

func CobraNew(data []string, title string) (p *tea.Program) {
	ds, _ := NewDataScope(data, false)
	return tea.NewProgram(ds, tea.WithAltScreen())
}

// displays the current page
func (s *DataScope) displayPage() string {
	start, end := s.pager.GetSliceBounds(len(s.data))
	data := s.data[start:end]
	// apply alterating color scheme
	var bldr strings.Builder
	var trueIndex int = start // index of full results, between start and end
	for _, d := range data {
		if trueIndex%2 == 0 {
			bldr.WriteString(evenEntryStyle.Render(d))
		} else {
			bldr.WriteString(oddEntryStyle.Render(d))
		}
		bldr.WriteRune('\n')
		trueIndex += 1
	}
	return bldr.String()
}

// generates a footer with the box+line and help keys
func (s *DataScope) footer() string {
	percent := infoStyle.Render(fmt.Sprintf("%3.f%%", s.vp.ScrollPercent()*100))
	line := "\n" + lipgloss.NewStyle().Foreground(stylesheet.PrimaryColor).Render(
		strings.Repeat("─", max(0, s.vp.Width-lipgloss.Width(percent))),
	)
	help := stylesheet.GreyedOutStyle.Render(
		fmt.Sprintf("%v page • %v scroll • tab: cycle • esc: quit",
			stylesheet.LeftRight, stylesheet.UpDown),
	)

	lineHelp := lipgloss.JoinVertical(lipgloss.Center, line, help)

	return lipgloss.JoinHorizontal(lipgloss.Center, lineHelp, percent) +
		"\n" +
		lipgloss.JoinVertical(lipgloss.Center, s.pager.View(), line)
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

var evenEntryStyle = lipgloss.NewStyle()
var oddEntryStyle = lipgloss.NewStyle().Foreground(stylesheet.SecondaryColor)

//#endregion
