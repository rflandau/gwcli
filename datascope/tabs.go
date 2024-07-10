package datascope

import (
	"errors"
	"fmt"
	"gwcli/clilog"
	"gwcli/connection"
	"gwcli/stylesheet"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/paginator"
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
// selected is the current index.
// fieldNum is the index of the field we are drawing
// (evaluated against selected for whether or not to draw the pip).
// field is the current value of the field that corresponds to fieldNum.
// fieldName is as it says on the tin.
// sty is the style to apply to fieldName.
// l/r Brack are the open/close brackets this boolean should use.
func viewBool(selected uint, fieldNum uint, field bool, fieldName string, sty lipgloss.Style, lBrack, rBrack rune) string {
	var checked rune = ' '
	if field {
		checked = '✓'
	}

	return fmt.Sprintf("%c%c%s%c %s", pip(selected, fieldNum), lBrack, sty.Render(string(checked)), rBrack, sty.Render(fieldName))
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
	dlappend
	dlfmtjson
	dlfmtcsv
	dlfmtraw
	dlpages
	dlhighBound
)

const outFilePerm = 0644

// TODO convert pages to records, for just downloading individual records (a la copying)
// likely must also support X-Y notation

type downloadTab struct {
	outfileTI textinput.Model // user input file to write to
	append    bool            // append to the outfile instead of truncating
	format    struct {
		json bool
		csv  bool
		raw  bool
	}

	pagesTI          textinput.Model // user input to select the pages to download
	selected         uint
	resultString     string // results of the previous download
	inputErrorString string // issues with current user input
}

// Initialize and return a DownloadTab struct suitable for representing the download option.
//
// ! JSON and CSV should not both be true. However, they can both be false.
// Setting both to true is undefined behavior.
func initDownloadTab(outfn string, append, json, csv bool) downloadTab {
	width := 20

	d := downloadTab{
		outfileTI: textinput.New(),
		append:    append,
		format: struct {
			json bool
			csv  bool
			raw  bool
		}{json: json, csv: csv, raw: false},
		pagesTI:  textinput.New(),
		selected: dloutfile,
	}

	// set raw if !(json or csv)
	if !json && !csv {
		d.format.raw = true
	}

	// initialize outfileTI
	d.outfileTI.Prompt = ""
	d.outfileTI.Width = width
	d.outfileTI.Placeholder = ""
	d.outfileTI.Focus()
	d.outfileTI.SetValue(outfn)

	// initialize pagesTI
	d.pagesTI.Prompt = ""
	d.pagesTI.Width = width
	d.pagesTI.Placeholder = "1,4,5"
	d.pagesTI.Blur()
	d.pagesTI.Validate = func(s string) error {
		for _, r := range s {
			if r == ',' || unicode.IsNumber(r) {
				continue
			}
			return errors.New("must be numeric")
		}
		return nil
	}

	return d
}

func updateDownload(s *DataScope, msg tea.Msg) tea.Cmd {
	if msg, ok := msg.(tea.KeyMsg); ok {
		s.download.inputErrorString = "" // clear input error on newest key message
		switch msg.Type {
		case tea.KeyUp:
			s.download.outfileTI.Blur()
			s.download.pagesTI.Blur()
			s.download.selected -= 1
			if s.download.selected <= dllowBound {
				s.download.selected = dlhighBound - 1
			}
			if s.download.selected == dloutfile {
				s.download.outfileTI.Focus()
			} else if s.download.selected == dlpages {
				s.download.pagesTI.Focus()
			}
			return nil
		case tea.KeyDown:
			s.download.outfileTI.Blur()
			s.download.pagesTI.Blur()
			s.download.selected += 1
			if s.download.selected >= dlhighBound {
				s.download.selected = dllowBound + 1
			}
			if s.download.selected == dloutfile {
				s.download.outfileTI.Focus()
			} else if s.download.selected == dlpages {
				s.download.pagesTI.Focus()
			}
			return nil
		case tea.KeySpace, tea.KeyEnter:
			if msg.Alt && msg.Type == tea.KeyEnter { // only accept alt+enter
				// gather and validate selections
				fn := strings.TrimSpace(s.download.outfileTI.Value())
				if fn == "" {
					str := "output file cannot be empty"
					s.download.inputErrorString = str
					return nil
				}
				res, success := s.dl(fn)
				s.download.resultString = res
				if !success {
					clilog.Writer.Error(res)
				} else {
					clilog.Writer.Info(res)
				}
				return nil
			}
			// handle booleans
			switch s.download.selected {
			case dlappend:
				s.download.append = !s.download.append
			case dlfmtjson:
				s.download.format.json = true
				if s.download.format.json {
					s.download.format.csv = false
					s.download.format.raw = false
				}
			case dlfmtcsv:
				s.download.format.csv = true
				if s.download.format.csv {
					s.download.format.json = false
					s.download.format.raw = false
				}
			case dlfmtraw:
				s.download.format.raw = true
				if s.download.format.raw {
					s.download.format.json = false
					s.download.format.csv = false
				}
			}
		}
	}

	// pass onto the TIs
	var cmds []tea.Cmd = make([]tea.Cmd, 2)
	s.download.outfileTI, cmds[0] = s.download.outfileTI.Update(msg)
	s.download.pagesTI, cmds[1] = s.download.pagesTI.Update(msg)

	return tea.Batch(cmds...)
}

// The actual download function that consumes the user inputs and creates a file
// based on the parameters.
// fn must not be the empty string.
// returns a string suitable for displaying to the user the result of the download
func (s *DataScope) dl(fn string) (result string, success bool) {
	var (
		err error
		f   *os.File // file path
	)

	baseErrorResultString := "Failed to save results to file: "

	// check append
	var flags int = os.O_CREATE | os.O_WRONLY
	if s.download.append {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	// attempt to open the file
	if f, err = os.OpenFile(fn, flags, outFilePerm); err != nil {
		return baseErrorResultString + err.Error(), false
	}
	defer f.Close()

	clilog.Writer.Debugf("Successfully opened file %v", f.Name())

	// branch on records-only or full download
	if strPages := strings.TrimSpace(s.download.pagesTI.Value()); strPages != "" {
		// specific records
		if err := dlrecords(f, strPages, &s.pager, s.data); err != nil {
			return baseErrorResultString + err.Error(), false
		}
		var word string = "Wrote"
		if s.download.append {
			word = "Appended"
		}

		return fmt.Sprintf("%v entries %v to %v", word, strPages, f.Name()), true
	}
	// whole file
	if err := connection.DownloadResults(s.search, f,
		s.download.format.json, s.download.format.csv); err != nil {
		return baseErrorResultString + err.Error(), false
	}

	return connection.DownloadQuerySuccessfulString(f.Name(), s.download.append), true
}

// helper record for dl.
// Downloads just the records specified.
func dlrecords(f *os.File, strPages string, pager *paginator.Model, results []string) error {
	var (
		pages []uint32
	)

	// explode and parse each page
	exploded := strings.Split(strPages, ",")
	for _, strpg := range exploded {
		// sanity check page
		pg, err := strconv.ParseUint(strpg, 10, 32)
		if err != nil {
			return fmt.Errorf("failed to parse page '%v':\n%v", strpg, err)
		}
		if pg > uint64(pager.TotalPages-1) {
			return fmt.Errorf(
				"page %v is outside the set of available pages [0-%v]",
				pg, pager.TotalPages-1)
		}
		// add it to the list of pages to download
		pages = append(pages, uint32(pg))
	}

	// allocate for the given # of pages
	data := make([]string, len(pages)*pager.PerPage)
	itemIndex := 0
	for _, pg := range pages {
		// fetch the data segment to append
		lBound, hBound := uint32(pager.PerPage)*pg, uint32(pager.PerPage)*(pg+1)-1
		clilog.Writer.Debugf("Page %v | lBound %v | hBound %v", pg, lBound, hBound)
		dslice := results[lBound:hBound]
		clilog.Writer.Debugf("dslice %v", dslice)

		// append each item in the segment
		for _, d := range dslice {
			data[itemIndex] = d
			itemIndex += 1
		}
	}
	for _, d := range data {
		if _, err := f.WriteString(d + "\n"); err != nil {
			return err
		}
	}

	return nil
}

// NOTE: the options section is mildly offset to the left.
// This is a byproduct of the invisible width of the TIs.
// There is probably a way to left-align each option but center on the longest width (the TIs),
// but that is left as an exercise for someone who cares.
func viewDownload(s *DataScope) string {
	var (
		sty lipgloss.Style = stylesheet.Header1Style
	)

	// create and join the options section elements
	options := lipgloss.JoinVertical(lipgloss.Left,
		sty.Render(" Output Path:"),
		fmt.Sprintf("%c%s", pip(s.download.selected, dloutfile), s.download.outfileTI.View()),
		viewBool(s.download.selected, dlappend, s.download.append, "Append?", sty, '[', ']'),
		sty.Render(" Format:"),
		viewBool(s.download.selected, dlfmtjson, s.download.format.json, "JSON", sty, '(', ')'),
		viewBool(s.download.selected, dlfmtcsv, s.download.format.csv, "CSV", sty, '(', ')'),
		viewBool(s.download.selected, dlfmtraw, s.download.format.raw, "RAW", sty, '(', ')'),
		sty.Render(" Pages:"),
		fmt.Sprintf("%c%s", pip(s.download.selected, dlpages), s.download.pagesTI.View()),
	)
	// center the options section
	hCenteredOptions := lipgloss.PlaceHorizontal(s.vp.Width, lipgloss.Center, options)

	// create the pages TI instructions
	pagesInst := lipgloss.NewStyle().
		Width(40).
		AlignHorizontal(lipgloss.Center).
		Italic(true).
		Render("Enter a comma-seperated list of pages to" +
			" download or leave it blank to download all results")

	// create the error/confirmation
	var end string // if an error is queued, display it
	if s.download.inputErrorString != "" {
		end = stylesheet.ErrStyle.Render(s.download.inputErrorString)
	} else {
		end = lipgloss.NewStyle().Foreground(stylesheet.AccentColor1).
			Render("Press alt+enter to confirm download.")
	}

	var downloaded string // if a download was previously performed, say so
	if s.download.resultString != "" {
		downloaded = "\n" +
			lipgloss.NewStyle().Foreground(stylesheet.AccentColor2).Render(s.download.resultString)
	}

	// join options, instructions, and end
	// centering and joining them independently allows the instructions to be wrapped and aligned
	// seperately, without altering the options section's alignment
	// once joined, vertically center the whole block
	return lipgloss.PlaceVertical(s.vp.Height, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center,
			hCenteredOptions, pagesInst, "\n", end, downloaded))
}

//#endregion

//#region schedule tab

type scheduleCursor = uint

const (
	schlowBound scheduleCursor = iota
	schname
	schdesc
	schcronfreq
	schhighBound
)

type scheduleTab struct {
	nameTI     textinput.Model
	descTI     textinput.Model
	cronfreqTI textinput.Model
}

func initScheduleTab() scheduleTab {
	sch := scheduleTab{
		nameTI:     stylesheet.NewTI(""),
		descTI:     stylesheet.NewTI(""),
		cronfreqTI: stylesheet.NewTI(""),
	}

	return sch
}

/*func updateSchedule(s *DataScope, msg tea.Msg) tea.Cmd {

}*/

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
