package datascope

import (
	"fmt"
	"gwcli/stylesheet"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
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
		viewFunc:   func(*DataScope) string { return compiledHelpString }}
	t[download] = tab{
		name:       "download",
		updateFunc: updateDownload,
		viewFunc:   viewDownload}
	t[schedule] = tab{
		name:       "schedule",
		updateFunc: func(*DataScope, tea.Msg) tea.Cmd { return nil },
		viewFunc:   func(*DataScope) string { return "" }}

	return t
}

// if this field is the selected field, returns the selection rune.
// otherwise, returns a space
func pip(selected, field uint) rune {
	if selected == field {
		return stylesheet.SelectionPrefix
	}
	return ' '
}

// Returns a string representing the current state of the given boolean value.
func viewBool(selected uint, fieldNum uint, field bool, fieldName string, sty lipgloss.Style) string {
	var checked rune = ' '
	if field {
		checked = '✓'
	}

	return fmt.Sprintf("%c[%s] %s\n", pip(selected, fieldNum), sty.Render(string(checked)), sty.Render(fieldName))
}

//#region results tab

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

//#endregion

//#region help tab

const cellWidth int = 25

var compiledHelpString string

// displays the available keys and useful information in a borderless table.
// This function rebuilds the help string, allowing it to only be regenerated when necessary
// (ex: a window size message) rather than every visible cycle.
func recompileHelp(s *DataScope) {
	// TODO split into 'all-tabs' keys and results-specific keys

	// we are hiding all border save for inter-row borders, so drop edge characters
	brdr := lipgloss.NormalBorder()
	brdr.MiddleLeft = ""
	brdr.MiddleRight = ""

	// Note the usage of width within these styles rather than the table's width.
	// Doing the reverse would cause long cells to truncate instead of wrap.
	// This method does *not* prevent truncation if the terminal is too small
	keyColumnStyle := lipgloss.NewStyle().Foreground(stylesheet.AccentColor1).
		MaxWidth(s.vp.Width / 2).Width(cellWidth)
	valueColumnStyle := lipgloss.NewStyle().MaxWidth(s.vp.Width / 2).Width(cellWidth)

	joinChar := ","

	tbl := table.New().
		Border(brdr).
		BorderRow(true).BorderColumn(false).
		BorderLeft(false).BorderRight(false).
		BorderTop(false).BorderBottom(false).
		BorderStyle(lipgloss.NewStyle().Foreground(stylesheet.TertiaryColor)).
		StyleFunc(func(row, col int) lipgloss.Style {
			if col == 0 {
				return keyColumnStyle
			}
			return valueColumnStyle
		})
	tbl.Rows(
		[][]string{
			{strings.Join(keys.cycleTabs.Keys(), joinChar), "cycle tables"},
			{strings.Join(keys.reverseCycleTabs.Keys(), joinChar), "reverse cycle tables"},
			{stylesheet.UpDown, "scroll page"},
			{stylesheet.LeftRight, "change page"},
			{strings.Join(keys.showTabs.Keys(), joinChar), "toggle tab visibility"},
			{"esc", "quit"},
		}...)

	// 'place' the table in the center of the *viewport*, horizontally and vertically
	compiledHelpString = lipgloss.Place(s.vp.Width, s.vp.Height,
		lipgloss.Center, lipgloss.Center, tbl.String())
}

//#endregion

//#region download tab

type downloadCursor = uint // current active item

const (
	dllowBound downloadCursor = iota
	dloutfile
	dlfmtjson
	dlfmtcsv
	dlfmtraw
	dlappend
	dlhighBound
)

type downloadTab struct {
	outfileTI textinput.Model // user input file to write to
	format    struct {
		json bool
		csv  bool
		raw  bool
	}
	append   bool // append to the outfile instead of truncating?
	selected uint
}

func initDownloadTab() downloadTab {
	d := downloadTab{
		format: struct {
			json bool
			csv  bool
			raw  bool
		}{json: false, csv: false, raw: true},
		outfileTI: textinput.New(),
		append:    false,
	}

	d.outfileTI.Prompt = ""
	d.outfileTI.Width = 20
	d.outfileTI.Placeholder = "(optional)"
	d.outfileTI.Blur()

	// start pointing to the outfile TI
	d.selected = dloutfile
	return d
}

func updateDownload(s *DataScope, msg tea.Msg) tea.Cmd {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case msg.Type == tea.KeyUp:
			s.download.selected -= 1
			if s.download.selected <= dllowBound {
				s.download.selected = dlhighBound - 1
			}
			if s.download.selected == dloutfile {
				s.download.outfileTI.Focus()
			} else {
				s.download.outfileTI.Blur()
			}
			return nil
		case msg.Type == tea.KeyDown:
			s.download.selected += 1
			if s.download.selected >= dlhighBound {
				s.download.selected = dllowBound + 1
			}
			if s.download.selected == dloutfile {
				s.download.outfileTI.Focus()
			} else {
				s.download.outfileTI.Blur()
			}
			return nil
		case msg.Alt && (msg.Type == tea.KeyEnter): // alt+enter
			// TODO download query to file iff outfile is populated
		case msg.Type == tea.KeySpace || msg.Type == tea.KeyEnter:
			switch s.download.selected {
			case dlappend:
				s.download.append = !s.download.append
			case dlfmtjson:
				s.download.format.json = !s.download.format.json
				if s.download.format.json {
					s.download.format.csv = false
					s.download.format.raw = false
				}
			case dlfmtcsv:
				s.download.format.csv = !s.download.format.csv
				if s.download.format.csv {
					s.download.format.json = false
					s.download.format.raw = false
				}
			case dlfmtraw:
				s.download.format.raw = !s.download.format.raw
				if s.download.format.raw {
					s.download.format.json = false
					s.download.format.csv = false
				}
			}
			return nil
		}
	}

	// pass onto the TI, if it is in focus
	var t tea.Cmd
	s.download.outfileTI, t = s.download.outfileTI.Update(msg)
	return t
}

func viewDownload(s *DataScope) string {
	var sb strings.Builder

	// styles
	var (
		sty lipgloss.Style = stylesheet.Header1Style
	)
	sb.WriteString(sty.Render("Output Path:") + "\n")
	sb.WriteString(
		fmt.Sprintf("%c%s\n", pip(s.download.selected, dloutfile), s.download.outfileTI.View()),
	)
	sb.WriteString(viewBool(s.download.selected, dlappend, s.download.append, "Append?", sty))
	sb.WriteRune('\n')
	sb.WriteString(sty.Render("Format:") + "\n")
	sb.WriteString(viewBool(s.download.selected, dlfmtjson, s.download.format.json, "JSON", sty))
	sb.WriteString(viewBool(s.download.selected, dlfmtcsv, s.download.format.csv, "CSV", sty))
	sb.WriteString(viewBool(s.download.selected, dlfmtraw, s.download.format.raw, "RAW", sty))
	sb.WriteRune('\n')
	sb.WriteString("Press alt+enter to confirm download.")

	// 'place' the options centered in the white space
	return lipgloss.Place(s.vp.Width, s.vp.Height,
		lipgloss.Center, lipgloss.Center,
		lipgloss.NewStyle().AlignHorizontal(lipgloss.Left).Render(sb.String()))

}

//#endregion

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
