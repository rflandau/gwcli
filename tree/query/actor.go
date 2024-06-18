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
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	grav "github.com/gravwell/gravwell/v3/client"
	"github.com/spf13/pflag"
)

//#region modes

// modes query model can be in
type mode int8

const (
	inactive  mode = iota // prepared, but not utilized
	prompting             // accepting user input
	quitting              // leaving prompt
	waiting               // search submitted; waiting for results
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

	spnr spinner.Model // wait spinner

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
	go func() { q.editor.ta.View() }()

	return q
}

func (q *query) Update(msg tea.Msg) tea.Cmd {
	switch q.mode {
	case quitting:
		return textarea.Blink
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

			// success
			clilog.Writer.Infof("Search succeeded. Fetching results (renderer %v)...", q.curSearch.RenderMod)
			q.mode = quitting
			results, err := connection.Client.GetTextResults(*q.curSearch, 0, 500)
			if err != nil {
				q.mode = prompting
				q.editor.err = err.Error()
				return textarea.Blink // we need to send a (any) msg to mother to trigger a redraw
			}

			clilog.Writer.Infof("%d results obtained", results.EntryCount)

			if q.output != nil {
				for _, e := range results.Entries {
					if _, err := q.output.Write(e.Data); err != nil {
						q.output.Close()
						return colorizer.ErrPrintf("Failed to write to %s: %v", q.output.Name(), err)
					}
					q.output.WriteString("\n")
				}
				q.output.Sync()
				q.output.Close()
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
	q.modifiers.selected = defaultModifSelection
	q.modifiers.durationTI.Reset()
	q.modifiers.outfileTI.Reset()
	q.modifiers.blur()
	q.modifiers.appendToFile = false

	// clear query fields
	q.curSearch = nil
	q.searchDone.Store(false)
	if q.output != nil {
		q.output.Close()
	}
	q.output = nil

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
	if d, err := localFS.GetDuration("duration"); err != nil {
		return "", []tea.Cmd{}, err
	} else if d != 0 {
		q.modifiers.durationTI.SetValue(d.String())
	}
	if o, err := localFS.GetString("output"); err != nil {
		return "", []tea.Cmd{}, err
	} else if o != "" {
		q.modifiers.outfileTI.SetValue(o)
	}

	// fetch and set a query, if given
	if tQry, err := FetchQueryString(&localFS, localFS.Args()); err != nil {
		return "", []tea.Cmd{}, err
	} else if tQry != "" {
		q.editor.ta.SetValue(tQry)
		// if the query is valid, submitQuery will place us directly into waiting mode
		return "", []tea.Cmd{q.submitQuery()}, nil
	}

	return "", nil, nil
}

//#region helper subroutines

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

	s, err := tryQuery(qry, duration)
	if err != nil {
		q.editor.err = err.Error()
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
