// action.Model (interactive) implementation of query
package query

import (
	"errors"
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
	"unicode"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
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

//#region editorView

// editorView represents the composable view box containing the query editor and any errors therein
type editorView struct {
	ta   textarea.Model
	err  string
	keys map[string]key.Binding
}

func initialEdiorView(height, width uint) editorView {
	ev := editorView{}

	// configure text area
	ev.ta = textarea.New()
	ev.ta.ShowLineNumbers = true
	ev.ta.Prompt = stylesheet.PromptPrefix
	ev.ta.SetWidth(int(width))
	ev.ta.SetHeight(int(height))
	ev.ta.Focus()
	// set up the help keys
	ev.keys = map[string]key.Binding{
		"submit": key.NewBinding(
			key.WithKeys("alt+enter"),
			key.WithHelp("alt+enter", "submit query"),
		)}

	return ev
}

func (ev *editorView) update(msg tea.Msg) (cmd tea.Cmd, submit bool) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, ev.keys["submit"]):
			if ev.ta.Value() == "" {
				// superfluous request
				ev.err = "empty request"
				// falls through to standard update
			} else {
				return nil, true
			}
		}
	}
	var t tea.Cmd
	ev.ta, t = ev.ta.Update(msg)
	return t, false
}

func (va *editorView) view() string {
	return fmt.Sprintf("Query:\n%s\n%s", va.ta.View(), va.err)
}

//#endregion editorView

//#region modifView

const selectionRune = '»'

type modifSelection = uint

const (
	lowBound modifSelection = iota
	duration
	outFile
	highBound
)

// modifView represents the composable view box containing all configurable features of the query
type modifView struct {
	width      uint
	height     uint
	selected   uint // tracks which modifier is currently active w/in this view
	durationTI textinput.Model
	outfileTi  textinput.Model
	keys       []key.Binding
}

// generate the second view to be composed with the query editor
func initialModifView(height, width uint) modifView {

	mv := modifView{
		width:    width,
		height:   height,
		selected: duration,
	}

	// build duration ti
	mv.durationTI = textinput.New()
	mv.durationTI.Width = int(width)
	mv.durationTI.Blur()
	mv.durationTI.Prompt = stylesheet.PromptPrefix
	mv.durationTI.Placeholder = "1h00m00s00ms00us00ns"
	mv.durationTI.Validate = func(s string) error {
		// checks that the string is composed of valid characters for duration parsing
		// (0-9 and h,m,s,u,n)
		// ! does not confirm that it is a valid duration!
		validChars := map[rune]interface{}{'h': nil, 'm': nil, 's': nil, 'u': nil, 'n': nil}
		for _, r := range s {
			if unicode.IsDigit(r) {
				continue
			}
			if _, f := validChars[r]; !f {
				return errors.New("only digits or the characters h, m, s, u, and n are allowed")
			}
		}
		return nil
	}

	// build outFile ti
	mv.outfileTi = textinput.New()
	mv.outfileTi.Width = int(width)
	mv.outfileTi.Blur()
	mv.outfileTi.Prompt = stylesheet.PromptPrefix

	return mv

}

// Unfocuses this view, blurring all text inputs
func (mv *modifView) blur() {
	mv.durationTI.Blur()
	mv.outfileTi.Blur()
}

func (mv *modifView) update(msg tea.Msg) []tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp:
			mv.selected -= 1
			if mv.selected <= lowBound {
				mv.selected = highBound - 1
			}
			mv.focusSelected()
		case tea.KeyDown:
			mv.selected += 1
			if mv.selected >= highBound {
				mv.selected = lowBound + 1
			}
			mv.focusSelected()
		}
	}
	var cmds []tea.Cmd = []tea.Cmd{}
	var t tea.Cmd
	mv.durationTI, t = mv.durationTI.Update(msg)
	if t != nil {
		cmds = append(cmds, t)
	}
	mv.outfileTi, t = mv.outfileTi.Update(msg)
	if t != nil {
		cmds = append(cmds, t)
	}

	return cmds
}

// Focuses the text input associated with the current selection, blurring all others
func (mv *modifView) focusSelected() {
	switch mv.selected {
	case duration:
		mv.durationTI.Focus()
		mv.outfileTi.Blur()
	case outFile:
		mv.durationTI.Blur()
		mv.outfileTi.Focus()
	default:
		clilog.Writer.Errorf("Failed to update modifier view focus: unknown selected field %d",
			mv.selected)
	}
}

func (mv *modifView) view() string {
	var bldr strings.Builder

	bldr.WriteString("Duration:\n")
	if mv.selected == duration {
		bldr.WriteRune(selectionRune)
	} else {
		bldr.WriteRune(' ')
	}
	bldr.WriteString(mv.durationTI.View() + "\n")

	bldr.WriteString("Output Path:\n")
	if mv.selected == outFile {
		bldr.WriteRune(selectionRune)
	} else {
		bldr.WriteRune(' ')
	}
	bldr.WriteString(mv.outfileTi.View() + "\n")

	return bldr.String()
}

//#region modifView

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

	spnr spinner.Model // wait spinner

	help struct {
		model help.Model
		keys  helpKeyMap
	}

	keys map[string]key.Binding // global keys, always active no matter the focused view

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
		duration:    defaultDuration,
	}

	// configure max dimensions
	q.width = 80
	q.height = 6

	q.editor = initialEdiorView(q.height, stylesheet.TIWidth)
	q.modifiers = initialModifView(q.height, q.width-stylesheet.TIWidth)

	q.focusedEditor = true

	q.keys = map[string]key.Binding{
		"cycle": key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "cycle view")),
		"quit":  key.NewBinding(key.WithHelp("esc", "return to navigation")),
	}

	// set up help
	q.help.model = help.New()
	q.help.keys = helpKeyMap{
		Cycle: key.NewBinding(
			key.WithKeys("tab"),
			key.WithKeys("tab", "cycle viewport"),
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
	go func() { q.editor.ta.View() }()

	return q
}

func (q *query) Update(msg tea.Msg) tea.Cmd {
	switch q.mode {
	case quitting:
		return textarea.Blink
	case inactive: // if inactive, bootstrap
		q.mode = prompting
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
			q.mode = quitting
			results, err := connection.Client.GetTextResults(*q.curSearch, 0, 500)
			if err != nil {
				q.mode = prompting
				q.editor.err = err.Error()
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

	// default, prompting mode

	keyMsg, isKeyMsg := msg.(tea.KeyMsg)

	// handle global keys
	if isKeyMsg {
		switch {
		case key.Matches(keyMsg, q.help.keys.Cycle):
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

	help := q.help.model.View(q.help.keys)

	return fmt.Sprintf("%s\n%s\n%s",
		lipgloss.JoinHorizontal(lipgloss.Top, q.editor.view(), q.modifiers.view()),
		help,
		blankOrSpnr)
}

func (q *query) Done() bool {
	return q.mode == quitting
}

func (q *query) Reset() error {
	// TODO update Reset to clear out views left and right
	q.mode = inactive
	localFS = initialLocalFlagSet()
	q.curSearch = nil
	q.editor.ta.Reset()
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
	// TODO take duration from second viewport
	var duration time.Duration = 1 * time.Hour
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

//#region help display

type helpKeyMap struct {
	Cycle  key.Binding
	Submit key.Binding // ctrl+enter
	//Help   key.Binding // '?'
	Quit key.Binding // esc
}

func (k helpKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Cycle, k.Submit, k.Quit}
}

// unused
func (k helpKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
}

//#endregion help display
