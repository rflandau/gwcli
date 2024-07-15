// Datascope is tabbed, scrolling viewport with a paginator built into the results view.
// It displays and manages results from a search.
// As the user pages through, the viewport automatically updates with the contents of the new page.
// The first tab contains the actual results, while the following tabs provide controls for
// downloading the results and scheduling the query
//
// Like busywait, this can be invoked by Cobra as a standalone tea.Model or as a child of an action
// spawned by Mother.
//
// ! There can only be one instance of Datascope running within a giving program; you should not
// compose datascopes from multiple searches. This is a caveat of the self-destructive goroutine
// used to keep the search object from aging out on the Gravwell backend.
package datascope

import (
	"errors"
	"gwcli/clilog"
	"gwcli/stylesheet"
	activesearchlock "gwcli/tree/query/datascope/ActiveSearchLock"
	"gwcli/utilities/killer"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
	grav "github.com/gravwell/gravwell/v3/client"
)

//#region keepAlive syncronization

const (
	pingFrequency = 50 * time.Second
	ageOut        = 5 * 60 * time.Second // 5 minutes
)

// Meant to be called as a goroutine that provides a heartbeat for the search id.
// This goroutine is self-destructive; if either:
//
//   - DS.Update() was last called too long ago (dictated by ageOut)
//
//     or
//
//   - the search ID this goroutine is tracking is replaced
//
// the goroutine will die on next wake.
func keepAlive(search *grav.Search) {
	/**
	 * A long-winded explanation of the problem this solves:
	 * This self-destructive goroutine-as-a-heartbeat is a necessary complilation over just having
	 * a gorutine that sleeps->pings->sleeps->(repeat ad nauseam) for the simple fact that DataScope
	 * (or any child or grandchild of Mother) does not know of its own death. When invoking actions
	 * non-interactively, we do not have to worry about this; goroutines die with their progenitor.
	 * However, a TUI is a single, long-running process; we must clean up after ourselves.
	 *
	 * More to the point, Mother is designed to always be able to re-assert control over her
	 * children, hence why she handles kill keys. The byproduct of this, however, is that we have no
	 * guarantee that a child action will be able to gracefully exit. Therefore, we need a mechanism
	 * to reap these goroutines (they aren't technically zombies in the same way a process would be,
	 * but the concept is the same).
	 *
	 * Mother could reap keepAlive, but 1) Mother is designed to be agnostic and does not care what
	 * her children are doing or if they have invoked DS and 2) we would need an event bus of sorts
	 * for mother to even have access to the context or channel with which to kill keepAlive.
	 *
	 * DS.Update() could ping for the search, but then frequency modulation becomes a problem.
	 * We can set a ticker so only the Update closest to the expiration of the ticker pings, but
	 * this adds weight to Update and only serves to limits pings; we have no way of guaranteeing
	 * that Update will be called frequently /enough/ for keepAlive.
	 *
	 * Thus, the best option is this: a goroutine that kills itself if it has reason to believe
	 * that its instance of DS is dead.
	 * Note that this solution still suffers from potentially too-infrequent Update calls. However,
	 * I believe this scenario is uncommon enough to be a risk worth taking; between the user
	 * navigating DS and the textinputs sending back blinks, we should be okay in most use cases.
	 *
	 */
	var mysid = search.ID
	for {
		if cursid := activesearchlock.GetSearchID(); cursid != mysid { // search ID changed
			clilog.Writer.Debugf("keepAlive: sid changed from %v to %v. Dying...", mysid, cursid)
			break
		}
		lastTS := activesearchlock.GetTS()
		oldestViableTS := time.Now().Unix() - int64(ageOut.Seconds())
		if oldestViableTS > lastTS { // last update was too long ago
			clilog.Writer.Debugf("keepAlive: search aged out (oldest viable %v > last %v). Dying...",
				oldestViableTS, lastTS)
			break
		}

		if err := search.Ping(); err != nil {
			clilog.Writer.Warnf("keepAlive: ping failed: %v", err)
			break
		}
		clilog.Writer.Debugf("pinged search %v", mysid)
		time.Sleep(pingFrequency)
	}
}

//#endregion

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

	// store data for keepAlive
	activesearchlock.SetSearchID(search.ID)
	activesearchlock.UpdateTS()
	// launch heartbeat gorotuine
	go keepAlive(search)

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
	// mother takes care of kill keys if she is running
	if !s.motherRunning {
		if kill := killer.CheckKillKeys(msg); kill != killer.None {
			clilog.Writer.Infof("Self-handled kill key, with kill type %v", kill)
			return s, tea.Batch(tea.Quit, tea.ExitAltScreen)
		}
	}

	// update the timestamp to keep the heartbeat going
	activesearchlock.UpdateTS()

	switch msg := msg.(type) {
	case tea.KeyMsg: // tab-agnostic keys
		switch {
		case key.Matches(msg, keys.showTabs):
			s.showTabs = !s.showTabs
			// recalculate height and update display
			s.setViewportHeight(s.rawWidth)
			return s, textinput.Blink
		case key.Matches(msg, keys.cycleTabs):
			s.activeTab += 1
			if s.activeTab >= uint(len(s.tabs)) {
				s.activeTab = 0
			}
			return s, textinput.Blink

		case key.Matches(msg, keys.reverseCycleTabs):
			if s.activeTab == 0 {
				s.activeTab = uint(len(s.tabs)) - 1
			} else {
				s.activeTab -= 1
			}
			return s, textinput.Blink
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
			s.ready = true
		} else { // just an update
			s.vp.Width = s.rawWidth
			s.setViewportHeight(msg.Width)
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

// Creates a new bubble tea program, in alt buffer mode, running only the DataScope.
// For use from Cobra.Run() subroutines.
// Start the returned program via .Run().
func CobraNew(data []string, search *grav.Search, outfn string,
	append, json, csv bool) (p *tea.Program, err error) {
	ds, _, err := NewDataScope(data, false, search, outfn, append, json, csv)
	if err != nil {
		return nil, err
	}
	return tea.NewProgram(ds, tea.WithAltScreen()), nil
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
