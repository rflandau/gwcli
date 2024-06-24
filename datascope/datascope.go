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
	vp            viewport.Model
	pager         paginator.Model
	ready         bool
	data          []string // complete set of data to be paged
	done          bool     // user has used a kill key
	Title         string   // displayed in the header box
	motherRunning bool     // without Mother's support, we need to handle killkeys and death alone
	margins       struct {
		header    string
		hdrHeight int
		footer    string
		ftrHeight int
	}
}

func (s DataScope) Init() tea.Cmd {
	// set header
	s.margins.header = func() string {
		title := viewportHeaderBoxStyle.Render(s.Title)
		line := strings.Repeat("─", max(0, s.vp.Width-lipgloss.Width(title)))
		dots := s.pager.View() + "\n"
		paragraph := lipgloss.JoinVertical(lipgloss.Center, dots, line)
		return lipgloss.JoinHorizontal(lipgloss.Center, title, paragraph)
	}()

	// set footer

	s.margins.footer = func() string {
		percent := infoStyle.Render(fmt.Sprintf("%3.f%%", s.vp.ScrollPercent()*100))
		line := strings.Repeat("─", max(0, s.vp.Width-lipgloss.Width(percent)))

		help := "h/l ←/→ page • q: quit\n"
		paragraph := lipgloss.JoinVertical(lipgloss.Center, line, help)
		return lipgloss.JoinHorizontal(lipgloss.Center, paragraph, percent)
	}()

	// pre-set heights
	s.margins.hdrHeight = lipgloss.Height(s.margins.header)
	s.margins.ftrHeight = lipgloss.Height(s.margins.footer)

	return nil
}

func (s DataScope) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			if s.motherRunning {
				s.done = true
				return s, tea.ExitAltScreen
			}
			return s, tea.Batch(tea.ExitAltScreen, tea.Quit)
		}
	case tea.WindowSizeMsg:

		marginHeight := s.margins.hdrHeight + s.margins.ftrHeight // extra space not showing content

		if !s.ready { // if we are not ready, use these dimensions to become ready
			s.vp = viewport.New(msg.Width, msg.Height-marginHeight)
			s.vp.YPosition = s.margins.hdrHeight
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

	return fmt.Sprintf("%s\n%s\n%s", s.margins.header, s.View(), s.margins.footer)
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

	return DataScope{
		pager:         p,
		ready:         false,
		data:          data,
		done:          false,
		Title:         title,
		motherRunning: motherRunning,
	}
}

// displays the current page
func (s DataScope) displayPage() string {
	start, end := s.pager.GetSliceBounds(len(s.data))
	imploded := strings.Join(s.data[start:end], "\n")
	return imploded
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
