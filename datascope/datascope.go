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
	"gwcli/clilog"
	"gwcli/stylesheet"
	"gwcli/utilities/killer"
	"os"
	"strings"

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
	Title         string   // displayed in the header box
	motherRunning bool     // without Mother's support, we need to handle killkeys and death alone
	hdrHeight     int
	ftrHeight     int
	marginHeight  int
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
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:

		if !s.ready { // if we are not ready, use these dimensions to become ready
			s.vp = viewport.New(msg.Width, msg.Height-s.marginHeight)
			s.vp.YPosition = s.hdrHeight
			s.vp.HighPerformanceRendering = false
			s.vp.SetContent(s.displayPage())
			s.ready = true
		} else { // just an update
			s.vp.Width = msg.Width
			s.vp.Height = msg.Height - s.marginHeight
		}
	}
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
	return s, tea.Sequence(cmds...)
}

func (s DataScope) View() string {
	if !s.ready {
		return "\nInitializing..."
	}
	return fmt.Sprintf("%s\n%s\n%s", s.header(), s.vp.View(), s.footer())
}

func CobraNew(data []string, title string) (p *tea.Program) {
	ds, _ := NewDataScope(data, false, title)
	return tea.NewProgram(ds, tea.WithAltScreen())
}

func NewDataScope(data []string, motherRunning bool, title string) (DataScope, tea.Cmd) {
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
		Title:         title,
		motherRunning: motherRunning,
	}
	// pre-set heights
	s.hdrHeight = lipgloss.Height(s.header())
	s.ftrHeight = lipgloss.Height(s.footer())
	s.marginHeight = s.hdrHeight + s.ftrHeight // extra space not showing content
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

// displays the current page
func (s *DataScope) displayPage() string {
	start, end := s.pager.GetSliceBounds(len(s.data))
	return strings.Join(s.data[start:end], "\n")
}

// generates a header with the box+line and page pips
func (s *DataScope) header() string {
	title := viewportHeaderBoxStyle.Render(s.Title)
	line := lipgloss.NewStyle().Foreground(stylesheet.PrimaryColor).Render(
		strings.Repeat("─", max(0, s.vp.Width-lipgloss.Width(title))),
	) + "\n"
	dotsLine := lipgloss.JoinVertical(lipgloss.Center, s.pager.View(), line)
	return lipgloss.JoinHorizontal(lipgloss.Center, title, dotsLine)
}

// generates a footer with the box+line and help keys
func (s *DataScope) footer() string {
	percent := infoStyle.Render(fmt.Sprintf("%3.f%%", s.vp.ScrollPercent()*100))
	line := "\n" + lipgloss.NewStyle().Foreground(stylesheet.PrimaryColor).Render(
		strings.Repeat("─", max(0, s.vp.Width-lipgloss.Width(percent))),
	)
	help := stylesheet.GreyedOutStyle.Render(
		fmt.Sprintf("%v page • %v scroll • esc: quit", stylesheet.LeftRight, stylesheet.UpDown),
	)

	lineHelp := lipgloss.JoinVertical(lipgloss.Center, line, help)

	return lipgloss.JoinHorizontal(lipgloss.Center, lineHelp, percent)
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
