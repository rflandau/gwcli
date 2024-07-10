// Datascope is tabbed, scrolling viewport with a paginator built into the results view.
// It displays arbitrary data, one page at a time, in the alt buffer.
// As the user pages through, the viewport automatically updates with the contents of the new page.
// The first tab contains the actual results, while
//
// Like busywait, this can be invoked for Cobra or for Mother.
package datascope

import (
	"errors"
	"fmt"
	"gwcli/clilog"
	"gwcli/stylesheet"
	"gwcli/utilities/killer"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
	grav "github.com/gravwell/gravwell/v3/client"
)

type DataScope struct {
	vp            viewport.Model
	pager         paginator.Model
	ready         bool
	data          []string // complete set of data to be paged
	motherRunning bool     // without Mother's support, we need to handle killkeys and death alone

	rawHeight int // usable height, as reported by the tty
	rawWidth  int // usabe width, as reported by the tty

	tabs      []tab // TODO junk tab array?
	showTabs  bool
	activeTab uint

	search *grav.Search // the search being displayed

	download downloadTab
	schedule scheduleTab
}

// Returns a new DataScope instance based on the given data array. If mother is running,
// this subroutine will launch her into the alt screen buffer and query the terminal for its size.
// outfn and append are optional; if outfn is given, the data will immediately be downloaded.
func NewDataScope(data []string, motherRunning bool, search *grav.Search, outfn string, append, json, csv bool) (DataScope, tea.Cmd, error) {
	// sanity check arguments
	if search == nil {
		return DataScope{}, nil, errors.New("search cannot be nil")
	}
	if len(data) == 0 {
		return DataScope{}, nil, errors.New("no data to display")
	}
	if json && csv {
		return DataScope{}, nil, errors.New("output format cannot be both JSON and CSV")
	}

	// set up backend paginator
	p := paginator.New()
	p.Type = paginator.Dots
	p.PerPage = 25
	p.ActiveDot = lipgloss.NewStyle().Foreground(stylesheet.FocusedColor).Render("•")
	p.InactiveDot = lipgloss.NewStyle().Foreground(stylesheet.UnfocusedColor).Render("•")
	p.SetTotalPages(len(data))

	s := DataScope{
		pager:         p,
		ready:         false,
		data:          data,
		motherRunning: motherRunning,
		download:      initDownloadTab(outfn, append, json, csv),
		schedule:      initScheduleTab(),
	}

	// set up tabs
	s.tabs = s.generateTabs()
	s.activeTab = results
	s.showTabs = true

	// save search
	s.search = search

	// if outfile was given, attempt automatic download
	if outfn != "" {
		res, success := s.dl(outfn)
		s.download.resultString = res
		if !success {
			clilog.Writer.Error(res)
		} else {
			clilog.Writer.Info(res)
		}
	}

	// mother does not start in alt screen, and thus requires manual measurements
	if motherRunning {
		return s, tea.Sequence(tea.EnterAltScreen, func() tea.Msg {
			w, h, err := term.GetSize(os.Stdin.Fd())
			if err != nil {
				clilog.Writer.Errorf("Failed to fetch terminal size: %v", err)
			}
			return tea.WindowSizeMsg{Width: w, Height: h}
		}), nil
	}

	return s, nil, nil
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
		switch {
		case key.Matches(msg, keys.showTabs):
			s.showTabs = !s.showTabs
			// recalculate height and update display
			s.setViewportHeight(s.rawWidth)
			s.vp.SetContent(s.displayPage())
			return s, nil
		case key.Matches(msg, keys.cycleTabs):
			s.activeTab += 1
			if s.activeTab >= uint(len(s.tabs)) {
				s.activeTab = 0
			}
		case key.Matches(msg, keys.reverseCycleTabs):
			if s.activeTab == 0 {
				s.activeTab = uint(len(s.tabs)) - 1
			} else {
				s.activeTab -= 1
			}
		}
	case tea.WindowSizeMsg:
		s.rawHeight = msg.Height
		s.rawWidth = msg.Width

		if !s.ready { // if we are not ready, use these dimensions to become ready
			s.vp = viewport.New(s.rawWidth, msg.Height)
			s.vp = viewport.Model{
				Width: s.rawWidth,
			}
			s.setViewportHeight(s.rawWidth)
			s.vp.MouseWheelDelta = 1
			s.vp.HighPerformanceRendering = false
			s.vp.SetContent(s.displayPage())
			s.ready = true
		} else { // just an update
			s.vp.Width = s.rawWidth
			s.setViewportHeight(msg.Width)
			s.vp.SetContent(s.displayPage())
		}

		recompileHelp(&s)
	}

	return s, s.tabs[s.activeTab].updateFunc(&s, msg)
}

func (s DataScope) View() string {
	if s.showTabs {
		return s.renderTabs(s.vp.Width) + "\n" + s.tabs[s.activeTab].viewFunc(&s)
	}
	return s.tabs[s.activeTab].viewFunc(&s)
}

func CobraNew(data []string, search *grav.Search, outfn string, append, json, csv bool) (p *tea.Program, err error) {
	ds, _, err := NewDataScope(data, false, search, outfn, append, json, csv)
	if err != nil {
		return nil, err
	}
	return tea.NewProgram(ds, tea.WithAltScreen()), nil
}

// displays the current page
func (s *DataScope) displayPage() string {
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
	return wrap(s.vp.Width, bldr.String())
}

// applies text wrapping to the given content. This is mandatory prior to SetContent, lest the text
// be clipped. It is a *possible* bug of the viewport bubble.
//
// (see:
// https://github.com/charmbracelet/bubbles/issues/479
// https://github.com/charmbracelet/bubbles/issues/56
// )
func wrap(width int, s string) string {
	return lipgloss.NewStyle().Width(width).Render(s)
}

var compiledShortHelp = stylesheet.GreyedOutStyle.Render(
	fmt.Sprintf("%v page • %v scroll • tab: cycle • 1-9: jump to tab • esc: quit",
		stylesheet.LeftRight, stylesheet.UpDown),
)

// generates a renderFooter with the box+line and help keys
func (s *DataScope) renderFooter(width int) string {
	percent := fmt.Sprintf("%3.f%%", s.vp.ScrollPercent()*100)
	line := "\n" + lipgloss.NewStyle().Foreground(stylesheet.PrimaryColor).Render(
		strings.Repeat("─", max(0, width-lipgloss.Width(percent))),
	)

	lineHelp := lipgloss.JoinVertical(lipgloss.Center, line, compiledShortHelp)

	pgr := fmt.Sprintf("%s %s",
		lipgloss.NewStyle().Foreground(stylesheet.FocusedColor).Width(3).Render(
			strconv.Itoa(s.pager.Page)),
		s.pager.View(),
	)

	return lipgloss.JoinVertical(lipgloss.Center,
		lipgloss.JoinHorizontal(lipgloss.Center, lineHelp, percent),
		pgr,
	)
}

// Sets the height of the viewport, using s.rawHeight minus the height of non-data segments
// (ex: the footer and tabs).
// Should be called after any changes to rawHeight, the tab header, or the footer.
func (s *DataScope) setViewportHeight(width int) {
	var tabHeight int
	if s.showTabs {
		tabHeight = lipgloss.Height(s.renderTabs(width))
	}
	footerHeight := lipgloss.Height(s.renderFooter(width))
	s.vp.Height = s.rawHeight - (tabHeight + footerHeight)

}

// #region styling
var evenEntryStyle = lipgloss.NewStyle()
var oddEntryStyle = lipgloss.NewStyle().Foreground(stylesheet.SecondaryColor)
var indexStyle = lipgloss.NewStyle().Foreground(stylesheet.AccentColor1)

//#endregion
