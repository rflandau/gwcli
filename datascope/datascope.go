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
	"strings"

	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

//#region For Cobra Usage

type DataScope struct {
	vp            viewport.Model
	pager         paginator.Model
	ready         bool
	data          []string // complete set of data to be paged
	done          bool     // user has used a kill key
	Title         string   // displayed in the header box
	motherRunning bool     // without Mother's support, we need to handle killkeys and death alone
	hdrHeight     int
	ftrHeight     int
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
		marginHeight := s.hdrHeight + s.ftrHeight // extra space not showing content

		if !s.ready { // if we are not ready, use these dimensions to become ready
			s.vp = viewport.New(msg.Width, msg.Height-marginHeight)
			s.vp.YPosition = s.hdrHeight
			s.vp.HighPerformanceRendering = false
			s.vp.SetContent(s.displayPage())
			s.ready = true
			cmds = append(cmds, tea.EnterAltScreen) // start the alt buffer
		} else { // just an update
			s.vp.Width = msg.Width
			s.vp.Height = msg.Height - marginHeight
		}
	}
	s.pager, cmd = s.pager.Update(msg)
	cmds = append(cmds, cmd)
	// pass the new content to the view
	s.vp.SetContent(s.displayPage())
	s.vp, cmd = s.vp.Update(msg)
	cmds = append(cmds, cmd)
	return s, tea.Sequence(cmds...)
}

func (s DataScope) View() string {
	if s.Done() {
		return "\nQuitting..."
	}
	if !s.ready {
		return "\nInitializing..."
	}
	return fmt.Sprintf("%s\n%s\n%s", s.header(), s.vp.View(), s.footer())
}

func (s DataScope) Done() bool {
	return s.done
}

func CobraNew(data []string, title string) (p *tea.Program) {
	return tea.NewProgram(NewDataScope(data, false, title))
}

//#endregion For Cobra Usage

func NewDataScope(data []string, motherRunning bool, title string) DataScope {
	// set up backend paginator
	p := paginator.New()
	p.Type = paginator.Dots
	p.PerPage = 30
	p.ActiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "235", Dark: "252"}).Render("•")
	p.InactiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "250", Dark: "238"}).Render("•")
	p.SetTotalPages(len(data))

	s := DataScope{
		pager:         p,
		ready:         false,
		data:          data,
		done:          false,
		Title:         title,
		motherRunning: motherRunning,
	}
	// pre-set heights
	s.hdrHeight = lipgloss.Height(s.header())
	s.ftrHeight = lipgloss.Height(s.footer())

	return s
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
	lineHelp := lipgloss.JoinVertical(lipgloss.Center,
		line,
		fmt.Sprintf("%v page • %v scroll • esc: quit", stylesheet.LeftRight, stylesheet.UpDown))

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
