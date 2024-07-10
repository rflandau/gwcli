package datascope

import (
	"errors"
	"fmt"
	"gwcli/clilog"
	"gwcli/connection"
	"gwcli/stylesheet"
	"gwcli/stylesheet/colorizer"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

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
		pagesTI:  textinput.New(), // TODO use stylesheet.NewTI()
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
		fmt.Sprintf("%s%s", colorizer.Pip(s.download.selected, dloutfile), s.download.outfileTI.View()),
		viewBool(s.download.selected, dlappend, s.download.append, "Append?", sty, '[', ']'),
		sty.Render(" Format:"),
		viewBool(s.download.selected, dlfmtjson, s.download.format.json, "JSON", sty, '(', ')'),
		viewBool(s.download.selected, dlfmtcsv, s.download.format.csv, "CSV", sty, '(', ')'),
		viewBool(s.download.selected, dlfmtraw, s.download.format.raw, "RAW", sty, '(', ')'),
		sty.Render(" Pages:"),
		fmt.Sprintf("%s%s", colorizer.Pip(s.download.selected, dlpages), s.download.pagesTI.View()),
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
