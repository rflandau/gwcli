// action.Model (interactive) implementation of query
package query

import (
	"fmt"
	"gwcli/action"
	"gwcli/busywait"
	"gwcli/clilog"
	"gwcli/connection"
	"gwcli/datascope"
	"gwcli/stylesheet"
	"gwcli/stylesheet/colorizer"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	grav "github.com/gravwell/gravwell/v3/client"
	"github.com/gravwell/gravwell/v3/client/types"
	"github.com/spf13/pflag"
)

//#region modes

// modes query model can be in
type mode int8

const (
	inactive   mode = iota // prepared, but not utilized
	prompting              // accepting user input
	quitting               // leaving prompt
	waiting                // search submitted; waiting for results
	displaying             // datascope is displaying results
)

//#endregion modes

// interactive model definition
type query struct {
	mode mode

	// total screen sizes for composing subviews
	width  uint
	height uint

	editor editorView

	modifiers modifView

	focusedEditor bool

	curSearch   *grav.Search // nil or ongoing/recently-completed search
	searchDone  atomic.Bool  // waiting thread has returned
	searchError chan error   // result to be fetched after SearchDone
	output      *os.File     // set once query is submitted, if outfile was set

	spnr  spinner.Model // wait spinner
	scope tea.Model     // interactively display data

	help help.Model

	keys []key.Binding // global keys, always active no matter the focused view

}

var Query action.Model = Initial()

func Initial() *query {
	q := &query{
		mode:        inactive,
		searchError: make(chan error),
		curSearch:   nil,
		spnr:        busywait.NewSpinner(),
	}

	// configure max dimensions
	q.width = 80
	q.height = 6

	q.editor = initialEdiorView(q.height, stylesheet.TIWidth)
	q.modifiers = initialModifView(q.height, q.width-stylesheet.TIWidth)

	q.focusedEditor = true

	q.keys = []key.Binding{
		key.NewBinding(key.WithKeys("tab"), // 0: cycle
			key.WithHelp("tab", "cycle view")),
		key.NewBinding(key.WithKeys("esc"), // [handled by mother]
			key.WithHelp("esc", "return to navigation")),
	}

	// set up help
	q.help = help.New()
	q.help.Width = int(q.width)

	BurnFirstView(q.editor.ta)

	return q
}

func (q *query) Update(msg tea.Msg) tea.Cmd {
	switch q.mode {
	case quitting:
		return textarea.Blink
	case displaying:
		if q.scope == nil {
			clilog.Writer.Errorf("query cannot be in display mode without a valid datascope")
			q.mode = quitting
		}
		// once we enter display mode, we do not leave until Mother kills us
		var cmd tea.Cmd
		q.scope, cmd = q.scope.Update(msg)
		return cmd
	case inactive: // if inactive, bootstrap
		q.mode = prompting
		q.editor.ta.Focus()
		q.focusedEditor = true
		return textarea.Blink
	case waiting: // display spinner and wait
		if q.searchDone.Load() {
			// search is done, check error, display results and exit
			if err := <-q.searchError; err != nil { // failure, return to text input
				q.editor.err = err.Error()
				q.mode = prompting
				var cmd tea.Cmd
				q.editor.ta, cmd = q.editor.ta.Update(msg)
				return cmd
			}

			// succcess
			if q.output != nil {
				defer q.output.Close()
			}

			var (
				results []types.SearchEntry
				err     error
			)
			if results, err = outputSearchResults(q.output, *q.curSearch,
				q.modifiers.json, q.modifiers.csv); err != nil {
				return colorizer.ErrPrintf("Failed to write to %s: %v", q.output.Name(), err)
			} else if results == nil {
				// already output to file, no more work needed
				q.mode = quitting
				return textinput.Blink
			}

			// display the output via datascope
			q.mode = displaying

			// output results as tea.Prints
			var data []string = make([]string, len(results))

			for i, r := range results {
				data[i] = string(r.Data)
			}

			s, cmd := datascope.NewDataScope(data, true)
			q.scope = s

			return cmd
		}
		// still waiting
		var cmd tea.Cmd
		q.spnr, cmd = q.spnr.Update(msg)
		return cmd
	}

	// default, prompting mode

	keyMsg, isKeyMsg := msg.(tea.KeyMsg)

	// handle global keys
	if isKeyMsg {
		switch {
		case key.Matches(keyMsg, q.keys[0]):
			q.switchFocus()
		}
	}

	// pass message to the active view
	var cmds []tea.Cmd
	if q.focusedEditor { // editor view active
		c, submit := q.editor.update(msg)
		if submit {
			return q.submitQuery()
		}
		cmds = []tea.Cmd{c}
	} else { // modifiers view active
		cmds = q.modifiers.update(msg)
	}

	return tea.Batch(cmds...)
}

func (q *query) View() string {
	if q.mode == displaying {
		return q.scope.View()
	}

	var blankOrSpnr string
	if q.mode == waiting { // if waiting, show a spinner instead of help
		blankOrSpnr = q.spnr.View()
	} else {
		blankOrSpnr = "\n"
	}

	var (
		viewKeys     []key.Binding
		editorView   string
		modifierView string
	)
	if q.focusedEditor {
		viewKeys = q.editor.keys
		editorView = stylesheet.Composable.Focused.Render(q.editor.view())
		modifierView = stylesheet.Composable.Unfocused.Render(q.modifiers.view())
	} else {
		viewKeys = q.modifiers.keys
		editorView = stylesheet.Composable.Unfocused.Render(q.editor.view())
		modifierView = stylesheet.Composable.Focused.Render(q.modifiers.view())
	}
	h := q.help.ShortHelpView(append(q.keys, viewKeys...))

	return fmt.Sprintf("%s\n%s\n%s",
		lipgloss.JoinHorizontal(lipgloss.Top, editorView, modifierView),
		h,
		blankOrSpnr)
}

func (q *query) Done() bool {
	return q.mode == quitting
}

func (q *query) Reset() error {
	// ! all inputs are blurred until user re-enters query later

	q.mode = inactive

	// reset editor view
	q.editor.ta.Reset()
	q.editor.err = ""
	q.editor.ta.Blur()
	// reset modifier view
	q.modifiers.reset()

	// clear query fields
	q.curSearch = nil
	q.searchDone.Store(false)
	if q.output != nil {
		q.output.Close()
	}
	q.output = nil
	q.scope = nil

	localFS = initialLocalFlagSet()

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
	if x, err := localFS.GetDuration("duration"); err != nil {
		return "", []tea.Cmd{}, err
	} else if x != 0 {
		q.modifiers.durationTI.SetValue(x.String())
	}
	if x, err := localFS.GetString("output"); err != nil {
		return "", []tea.Cmd{}, err
	} else if x != "" {
		q.modifiers.outfileTI.SetValue(x)
	}
	if x, err := localFS.GetString("name"); err != nil {
		return "", []tea.Cmd{}, err
	} else if x != "" {
		q.modifiers.schedule.nameTI.SetValue(x)
		//q.modifiers.schedule.enabled = true
	}
	if x, err := localFS.GetString("description"); err != nil {
		return "", []tea.Cmd{}, err
	} else if x != "" {
		q.modifiers.schedule.descTI.SetValue(x)
		//.modifiers.schedule.enabled = true
	}
	if x, err := localFS.GetString("schedule"); err != nil {
		return "", []tea.Cmd{}, err
	} else if x != "" {
		q.modifiers.schedule.descTI.SetValue(x)
		q.modifiers.schedule.enabled = true
	}

	// fetch and set a query, if given
	if tQry, err := fetchQueryString(&localFS, localFS.Args()); err != nil {
		return "", []tea.Cmd{}, err
	} else if tQry != "" {
		q.editor.ta.SetValue(tQry)
		// if the query is valid, submitQuery will place us directly into waiting mode
		return "", []tea.Cmd{q.submitQuery()}, nil
	}

	return "", nil, nil
}

//#region helper subroutines

// Gathers information across both views and initiates the search, placing the model into a waiting
// state. A seperate goroutine, initialized here, waits on the search, allowing this thread to
// display a spinner.
// Corrollary to `outputSearchResults` (connected via `case waiting` in Update()).
func (q *query) submitQuery() tea.Cmd {
	qry := q.editor.ta.Value() // clarity

	clilog.Writer.Infof("Submitting query '%v'...", qry)

	// fetch modifiers from alternative view
	var (
		duration time.Duration
		err      error
	)
	if d := strings.TrimSpace(q.modifiers.durationTI.Value()); d != "" {
		duration, err = time.ParseDuration(q.modifiers.durationTI.Value())
		if err != nil {
			q.editor.err = err.Error()
			return nil
		}
	} else {
		duration = defaultDuration
	}

	// prepare file for output and associate it to the query struct
	if fn := strings.TrimSpace(q.modifiers.outfileTI.Value()); fn != "" {
		q.output, err = openFile(fn, q.modifiers.appendToFile)
		if err != nil {
			q.editor.err = err.Error()
			return nil
		}
	} else { // do not output to file
		q.output = nil
	}

	// fetch schedule
	var sch *schedule = nil
	if q.modifiers.schedule.enabled {
		sch = &schedule{}
		sch.name = q.modifiers.schedule.nameTI.Value()
		sch.desc = q.modifiers.schedule.descTI.Value()
		sch.cronfreq = q.modifiers.schedule.cronfreqTI.Value()
	}

	s, schID, err := tryQuery(qry, -duration, sch)
	if err != nil {
		q.editor.err = err.Error()
		return nil
	}
	if schID != 0 { // if we scheduled a query, just exit
		q.mode = quitting
		return tea.Printf("Scheduled search (ID: %v)", schID)
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

func (q *query) switchFocus() {
	q.focusedEditor = !q.focusedEditor
	if q.focusedEditor { // disable viewB interactions
		q.modifiers.blur()
		q.editor.ta.Focus()
	} else { // disable query editor interaction
		q.editor.ta.Blur()
		q.modifiers.focusSelected()
	}
}

//#endregion helper subroutines

func BurnFirstView(ta textarea.Model) {

	/**
	 * Omitting this superfluous view outputs rgb control characters to the *first* instance of the
	 * query editor.
	 */
	_ = ta.View()

	/**
	 * A deeper dive:
	 * Formerly, Actions, particularly actions with TextArea/TextInputs hung the first time one was
	 * invoked each time the program launched. They eventually redrew, fixing the issue, but it
	 * could take quite a while.
	 * What was weird was that it was *not* that each one hung, but that the first hung and then all
	 * actions thereafter were fine. In other words, it was either related to a costly
	 * initialization in TA/TIs or not properly triggering redraws (by not sending tea.Cmds were we
	 * should be).
	 * The errant view call above was wrapped in a goroutine
	 * (`go func() { q.editor.ta.View() }()`)
	 * and it paid the startup cost in a way invisible to the user so the UX was seamless.
	 * Some optimizations and reworks later, and I figued out that the hang/redraw issue was likely
	 * due to missing tea.Cmds (the latter of the possibilities above).
	 *
	 * However, I also discovered that the go .view instruction was causing garbage (rgb control
	 * characters) to be output to the terminal if Mother was not invoked to catch the characters.
	 * This caused *some* non-interactive commands to output garbage to the users terminal or, worst
	 * case, break older shells (such as `sh`).
	 *
	 * The RGB control characters issue still persists and eliminating the above call causes garbage
	 * to appear in the first, interactive call to query.
	 * I have looked into the issue and it seems to stem from termenv.
	 * These characters are requested by termenv on startup to determine the capabilities of the
	 * terminal, but can be output to the terminal if term latency is too high.
	 * Supposedly this issue was fixed in termenv in [2021](https://github.com/muesli/termenv/pull/27).
	 * This means one of two things: the issue is not as resovled as it seems or, more likely, we or
	 * lipgloss are doing something ill-advised that causes these characters to not be collected by
	 * termenv properly.
	 *
	 * I would love to know what the issue is and hope to dedicate time to delving into termenv and
	 * lipgloss to investigate, but termenv is a doozy and my time is better spent elsewhere, as
	 * this band-aid is doing its job for minimal technical debt.
	 */

}
