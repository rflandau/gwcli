// action.Model (interactive) implementation of query
package query

import (
	"fmt"
	"gwcli/action"
	"gwcli/busywait"
	"gwcli/clilog"
	"gwcli/connection"
	"gwcli/stylesheet"
	"gwcli/stylesheet/colorizer"
	"os"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	grav "github.com/gravwell/gravwell/v3/client"
	"github.com/spf13/pflag"
)

type mode int8

const (
	inactive  mode = iota // prepared, but not utilized
	prompting             // accepting user input
	quitting              // leaving prompt
	waiting               // search submitted; waiting for results
)

type query struct {
	mode  mode
	error string // errors are mostly cleared by the next key input

	curSearch   *grav.Search // nil or ongoing/recently-completed search
	searchDone  atomic.Bool  // waiting thread has returned
	searchError chan error   // result to be fetched after SearchDone

	spnr spinner.Model // wait spinner

	help struct {
		model help.Model
		keys  helpKeyMap
	}
	ta textarea.Model

	outFile *os.File

	duration time.Duration
}

var Query action.Model = Initial()

func Initial() *query {
	q := &query{
		mode:        inactive,
		searchError: make(chan error),
		curSearch:   nil,
		spnr:        busywait.NewSpinner(),
		error:       "",
		duration:    defaultDuration,
	}

	// configure text area
	q.ta = textarea.New()
	q.ta.ShowLineNumbers = true
	q.ta.Prompt = "->"
	q.ta.SetWidth(stylesheet.TIWidth)
	q.ta.SetHeight(5)
	q.ta.Focus()

	// set up help
	q.help.model = help.New()
	q.help.keys = helpKeyMap{
		Submit: key.NewBinding(
			key.WithKeys("alt+enter"),
			key.WithHelp("alt+enter", "submit query"),
		),
		Quit: key.NewBinding(
			key.WithHelp("esc", "return to navigation"),
		),
	}

	// Actions, particularly actions with Help and TextArea/TextInputs hang the first time one is
	// called every time the program is launched. They eventually redraw, fixing the issue, but
	// sometimes require a msg (generally in the form of user input) to redraw.
	// What is weird is that it is *not* that each one hangs, but that the first hangs and then all
	// actions are fine after that.
	// This errant view call gets that out of the way in the back so the UX is seamless.
	// This is likely due to some lazy initialization within Bubble Tea OR (more likely) us not
	// sending a msg (ex: a blink) back to Mother during handoff. Not clear if this message should
	// be coming from Mother herself or from the recently in-control child.
	// TODO figure out why this works and what the proper fix is
	go func() { q.ta.View() }()

	return q
}

func (q *query) Update(msg tea.Msg) tea.Cmd {
	switch q.mode {
	case quitting:
		return textarea.Blink
	case inactive:
		q.mode = prompting
		return textarea.Blink
	case waiting: // display spinner and wait
		if q.searchDone.Load() {
			// search is done, check error, display results and exit
			if err := <-q.searchError; err != nil { // failure, return to text input
				q.error = err.Error()
				q.mode = prompting
				var cmd tea.Cmd
				q.ta, cmd = q.ta.Update(msg)
				return cmd
			}

			// success
			q.mode = quitting
			results, err := connection.Client.GetTextResults(*q.curSearch, 0, 500)
			if err != nil {
				q.mode = prompting
				q.error = err.Error()
				return textarea.Blink // we need to send a (any) msg to mother to trigger a redraw
			}

			if q.outFile != nil {
				for _, e := range results.Entries {
					if _, err := q.outFile.Write(e.Data); err != nil {
						return colorizer.ErrPrintf("Failed to write to %s: %v", q.outFile.Name(), err)
					}
					q.outFile.WriteString("\n")
				}
				return textarea.Blink // we need to send a (any) msg to mother to trigger a redraw
			}

			// print to screen
			var cmds []tea.Cmd = make([]tea.Cmd, results.EntryCount)

			for i, e := range results.Entries {
				cmds[i] = tea.Printf("%s\n", e.Data)
			}

			return tea.Sequence(cmds...)
		}
		// still waiting
		var cmd tea.Cmd
		q.spnr, cmd = q.spnr.Update(msg)
		return cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		q.error = "" // clear out the error
		if key.Matches(msg, q.help.keys.Submit) {
			if q.ta.Value() == "" {
				// superfluous request
				q.error = "empty request"
				// falls through to standard update
			} else {
				return q.submitQuery()
			}
		}
	}

	var cmd tea.Cmd
	q.ta, cmd = q.ta.Update(msg)
	return cmd
}

func (q *query) View() string {
	var errOrSpnr string
	if q.mode == waiting { // if waiting, show a spinner instead of help
		errOrSpnr = q.spnr.View()
	} else {
		errOrSpnr = q.error
	}

	help := q.help.model.View(q.help.keys)
	ta := q.ta.View()

	return fmt.Sprintf("Query:\n%s\n%s\n%s", ta, help, errOrSpnr)
}

func (q *query) Done() bool {
	return q.mode == quitting
}

func (q *query) Reset() error {
	q.mode = inactive
	q.error = ""
	localFS = initialLocalFlagSet()
	q.curSearch = nil
	q.ta.Reset()
	q.duration = defaultDuration
	q.searchDone.Store(false)
	return nil
}

// Consume flags and associated them to the local flagset
func (q *query) SetArgs(_ *pflag.FlagSet, tokens []string) (string, []tea.Cmd, error) {
	// parse the tokens agains the local flagset
	err := localFS.Parse(tokens)
	if err != nil {
		return "", []tea.Cmd{}, err
	}

	// fetch and set normal flags
	if d, err := localFS.GetDuration("duration"); err != nil {
		return "", []tea.Cmd{}, err
	} else if d != 0 {
		q.duration = d
	}
	if q.outFile, err = openOutFile(&localFS); err != nil {
		return "", []tea.Cmd{}, err
	}

	// fetch and set a query, if given
	if tQry, err := FetchQueryString(&localFS, localFS.Args()); err != nil {
		return "", []tea.Cmd{}, err
	} else if tQry != "" {
		q.ta.SetValue(tQry)
		// if the query is valid, submitQuery will place us directly into waiting mode
		return "", []tea.Cmd{q.submitQuery()}, nil
	}

	return "", nil, nil
}

//#region interactive-specific helper subroutines

func (q *query) submitQuery() tea.Cmd {
	qry := q.ta.Value() // clarity

	clilog.Writer.Infof("Submitting query '%v'...", qry)
	// TODO take duration from second viewport
	var duration time.Duration = 1 * time.Hour
	s, err := tryQuery(qry, duration)
	if err != nil {
		q.error = err.Error()
		return nil
	}
	// spin up a goroutine to wait on the search while we show a spinner
	go func() {
		err := connection.Client.WaitForSearch(s)
		// notify we are done and buffer the error for retrieval
		q.searchDone.Store(true)
		q.searchError <- err
	}()

	q.curSearch = &s
	q.mode = waiting
	return q.spnr.Tick // start the wait spinner
}

//#endregion

//#region help display

type helpKeyMap struct {
	Submit key.Binding // ctrl+enter
	//Help   key.Binding // '?'
	Quit key.Binding // esc
}

func (k helpKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Submit, k.Quit}
}

// unused
func (k helpKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
}
