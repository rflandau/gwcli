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
	dlrecords
	dlhighBound
)

const outFilePerm = 0644

type downloadTab struct {
	outfileTI textinput.Model // user input file to write to
	append    bool            // append to the outfile instead of truncating
	format    struct {
		enabled bool
		json    bool
		csv     bool
		raw     bool
	}

	recordsTI        textinput.Model // user input to select the pages to download
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
			enabled bool
			json    bool
			csv     bool
			raw     bool
		}{enabled: true, json: json, csv: csv, raw: false},
		recordsTI: textinput.New(), // TODO use stylesheet.NewTI()
		selected:  dloutfile,
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

	// initialize recordsTI
	d.recordsTI.Prompt = ""
	d.recordsTI.Width = width
	d.recordsTI.Placeholder = "1,4,740"
	d.recordsTI.Blur()
	d.recordsTI.Validate = func(s string) error {
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
			cycleUp(&s.download)
			return nil
		case tea.KeyDown:
			cycleDown(&s.download)
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
	s.download.recordsTI, cmds[1] = s.download.recordsTI.Update(msg)

	// if recordsTI has input, disable format section
	if strings.TrimSpace(s.download.recordsTI.Value()) != "" {
		s.download.format.enabled = false
	} else {
		s.download.format.enabled = true
	}

	return tea.Batch(cmds...)
}

// Cycle Up steps once up the list of options (defined by the downloadCursor enumerations),
// skipping the format section if it is disabled and looping to the last option if the user cycles
// up while on the first.
func cycleUp(dl *downloadTab) {
	dl.outfileTI.Blur()
	dl.recordsTI.Blur()
	dl.selected -= 1
	// if the format section is disabled, skip its elements
	if !dl.format.enabled {
		switch dl.selected {
		case dlfmtjson, dlfmtcsv, dlfmtraw:
			dl.selected = dlappend
		} // if no format elements are selection do nothing
	}
	if dl.selected <= dllowBound {
		dl.selected = dlhighBound - 1
	}
	if dl.selected == dloutfile {
		dl.outfileTI.Focus()
	} else if dl.selected == dlrecords {
		dl.recordsTI.Focus()
	}
}

// See cycleUp()
func cycleDown(dl *downloadTab) {
	dl.outfileTI.Blur()
	dl.recordsTI.Blur()
	dl.selected += 1
	// if the format section is disabled, skip its elements
	if !dl.format.enabled {
		switch dl.selected {
		case dlfmtjson, dlfmtcsv, dlfmtraw:
			dl.selected = dlrecords
		} // if no format elements are selection do nothing
	}
	if dl.selected >= dlhighBound {
		dl.selected = dllowBound + 1
	}
	if dl.selected == dloutfile {
		dl.outfileTI.Focus()
	} else if dl.selected == dlrecords {
		dl.recordsTI.Focus()
	}
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
	if strRecords := strings.TrimSpace(s.download.recordsTI.Value()); strRecords != "" {
		// specific records
		records, err := dlrecordsOnly(f, strRecords, s.data)
		if err != nil {
			return baseErrorResultString + err.Error(), false
		}
		var word string = "Wrote"
		if s.download.append {
			word = "Appended"
		}
		return fmt.Sprintf("%v entries %v to %v", word, records, f.Name()), true
	}
	// whole file
	if err := connection.DownloadResults(s.search, f,
		s.download.format.json, s.download.format.csv); err != nil {
		return baseErrorResultString + err.Error(), false
	}

	return connection.DownloadQuerySuccessfulString(f.Name(), s.download.append), true
}

// helper record for dl.
// Writes just the records specified in the comma-seperated list strRecords to the file f.
// Returns the list of record numbers whose values were written or an error
func dlrecordsOnly(f *os.File, strRecords string, data []string) ([]uint32, error) {
	exploded := strings.Split(strRecords, ",")
	var writtenRecords []uint32 = make([]uint32, len(exploded))
	var i int = 0
	for _, strRec := range exploded {
		// sanity check record
		if strings.TrimSpace(strRec) == "" {
			continue
		}
		var rec uint32
		if n, err := strconv.ParseUint(strRec, 10, 32); err != nil {
			return nil, fmt.Errorf("failed to parse record '%v':\n%v", strRec, err)
		} else {
			rec = (uint32(n))
		}
		rec -= 1 // decrement, as the user sees records starting at 1, instead of 0

		totalRecords := uint32(len(data))
		if rec > totalRecords {
			return nil, fmt.Errorf(
				"record %v is outside the set of available records [0-%v]",
				rec, totalRecords)
		}

		// requested number is in good condition; write is data to the file
		f.WriteString(data[rec] + "\n")
		writtenRecords[i] = rec + 1 // increment back to the user-displayed record #
		i++
	}

	return writtenRecords, nil
}

func viewDownload(s *DataScope) string {
	sel := s.download.selected // brevity
	width := s.download.outfileTI.Width + 5

	var ( // shared styles
		titleSty    lipgloss.Style = stylesheet.Header1Style
		subtitleSty                = stylesheet.Header2Style
		lcolAligner lipgloss.Style = lipgloss.NewStyle().Width(width).AlignHorizontal(lipgloss.Right).PaddingRight(1)
		rcolAligner lipgloss.Style = lipgloss.NewStyle().Width(width).AlignHorizontal(lipgloss.Left)
	)

	tabDesc := tabDescStyle(s.vp.Width).Render("Download all data in your preferred format or" +
		" cherry-pick specific records by their index.")

	prime := outputFormatSegment(titleSty, subtitleSty, lcolAligner, rcolAligner, sel, &s.download)

	recs := recordSegment(titleSty, lcolAligner, rcolAligner, sel, &s.download)

	return lipgloss.Place(s.vp.Width, s.vp.Height,
		lipgloss.Center, verticalPlace,
		lipgloss.JoinVertical(lipgloss.Center,
			tabDesc,
			prime,
			"",
			recs,
			"",
			submitString(s.download.inputErrorString, s.download.resultString),
		),
	)
}

// helper subroutine for viewDownload.
// Generates output and format segments and joins them together.
func outputFormatSegment(titleSty, subtitleSty, lcolAligner, rcolAligner lipgloss.Style,
	selected downloadCursor, dl *downloadTab) string {
	// generate output segement

	var ( // left column strings
		outputStr = fmt.Sprintf("%s%s",
			colorizer.Pip(selected, dloutfile), titleSty.Render("Output Path:"))
		appendStr = fmt.Sprintf("%s%s",
			colorizer.Pip(selected, dlappend), subtitleSty.Render("Append?"))
	)

	l := lipgloss.JoinVertical(lipgloss.Right,
		lcolAligner.Render(outputStr),
		lcolAligner.Render(appendStr),
	)

	var (
		outputTIStr = dl.outfileTI.View()
		appendBox   = colorizer.Checkbox(dl.append)
	)

	r := lipgloss.JoinVertical(lipgloss.Left,
		rcolAligner.Render(outputTIStr),
		rcolAligner.Render(appendBox))

	// conjoin output pieces
	outputSeg := lipgloss.JoinHorizontal(lipgloss.Center, l, r)

	// if records is set, do not display the format section
	if !dl.format.enabled {
		return outputSeg
	}

	// generate format segment
	var ( // format segment left column elements
		jsonStr = fmt.Sprintf("%s%s",
			colorizer.Pip(selected, dlfmtjson), subtitleSty.Render("JSON"))
		csvStr = fmt.Sprintf("%s%s",
			colorizer.Pip(selected, dlfmtcsv), subtitleSty.Render("CSV"))
		rawStr = fmt.Sprintf("%s%s",
			colorizer.Pip(selected, dlfmtraw), subtitleSty.Render("raw"))
	)

	var ( // format segment right column elements
		jsonBox = colorizer.Radiobox(dl.format.json)
		csvBox  = colorizer.Radiobox(dl.format.csv)
		rawBox  = colorizer.Radiobox(dl.format.raw)
	)

	// conjoin format pieces
	formatSeg := lipgloss.JoinHorizontal(lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Right,
			lcolAligner.Render(jsonStr),
			lcolAligner.Render(csvStr),
			lcolAligner.Render(rawStr)),
		lipgloss.JoinVertical(lipgloss.Left,
			rcolAligner.Render(jsonBox),
			rcolAligner.Render(csvBox),
			rcolAligner.Render(rawBox)),
	)

	return lipgloss.JoinVertical(lipgloss.Center,
		outputSeg,
		titleSty.Render("Format"),
		formatSeg)
}

func recordSegment(titleSty, lcolAligner, rcolAligner lipgloss.Style,
	selected downloadCursor, dl *downloadTab) string {
	// grey-out records if the TI is empty
	recSty := titleSty
	if strings.TrimSpace(dl.recordsTI.Value()) == "" {
		recSty = stylesheet.GreyedOutStyle
	}

	recs := lipgloss.JoinHorizontal(lipgloss.Center,
		lcolAligner.Render(fmt.Sprintf("%s%s",
			colorizer.Pip(selected, dlrecords), recSty.Render("Record Numbers:"))),
		rcolAligner.Render(dl.recordsTI.View()),
	)

	recordsDesc := lipgloss.NewStyle().Width(40).AlignHorizontal(lipgloss.Center).Italic(true).
		Render("OPTIONAL:\n" + "Enter a comma-seperated list of records to download just those records," +
			" instead of the whole file.")
	return lipgloss.JoinVertical(lipgloss.Center,
		recordsDesc,
		recs)
}

//#endregion