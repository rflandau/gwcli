package query

import (
	"errors"
	"fmt"
	"gwcli/action"
	"gwcli/clilog"
	cobraspinner "gwcli/cobra_spinner"
	"gwcli/connection"
	"gwcli/treeutils"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	grav "github.com/gravwell/gravwell/v3/client"
	"github.com/gravwell/gravwell/v3/client/types"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// the Gravwell client can only consume time formatted as follows
const timeFormat = "2006-01-02T15:04:05.999999999Z07:00"

var (
	ErrSuperfluousQuery = "query is empty and therefore ineffectual"
)

var localFS pflag.FlagSet

//#region command/action set up

func GenerateAction() action.Pair {
	cmd := treeutils.NewActionCommand("query", "submit a query",
		"Generate and send a query to the remote server. Results can be received via this cli or later on the web GUI.\n"+
			"All arguments after `query` will be passed to the instance as the string to query.", []string{}, run)

	localFS = initialLocalFlagSet()

	cmd.Example = "./gwcli -u USERNAME -p PASSWORD query tag=gravwell"

	cmd.Flags().AddFlagSet(&localFS)

	cmd.MarkFlagsRequiredTogether("name", "description", "schedule")

	return treeutils.GenerateAction(cmd, Query)
}

func initialLocalFlagSet() pflag.FlagSet {
	fs := pflag.FlagSet{}

	fs.DurationP("duration", "t", time.Hour*1, "the amount of time over which the query should be run.\n"+"Default: 1h")
	fs.StringP("reference", "r", "", "a reference to a query library item to execute instead of a provided query.")
	fs.StringP("output", "o", "", "file to write results to.")

	// scheduled searches
	fs.StringP("name", "n", "", "name for a scheduled search")
	fs.StringP("description", "d", "", "(flavour) description")
	fs.StringP("schedule", "s", "", "schedule this search to be run at a later date, over the given duration.")

	return fs
}

//#endregion

//#region cobra command

func run(cmd *cobra.Command, args []string) {
	var err error

	// fetch required flags
	duration, err := cmd.Flags().GetDuration("duration")
	if err != nil {
		clilog.TeeError(cmd.ErrOrStderr(), err.Error())
		return
	}

	if schedule, err := cmd.Flags().GetString("schedule"); err != nil {
		clilog.TeeError(cmd.ErrOrStderr(), err.Error())
		return
	} else if schedule != "" {
		var name, description, schedule string
		clilog.Writer.Infof("Scheduling search %v, %v, %v... (NYI)",
			name, description, schedule)
		// TODO implement scheduled searches
		return
	}

	q, err := FetchQueryString(cmd, args)
	if err != nil {
		clilog.TeeError(cmd.ErrOrStderr(), err.Error())
		return
	}

	s, err := tryQuery(q, duration)
	if err != nil {
		clilog.TeeError(cmd.ErrOrStderr(), err.Error())
		return
	}

	// spin up a spinner
	spnrP := cobraspinner.New()
	go func() {
		if err := connection.Client.WaitForSearch(s); err != nil {
			clilog.TeeError(cmd.ErrOrStderr(), err.Error())
			return
		}
		spnrP.Quit()
	}()

	if _, err := spnrP.Run(); err != nil {
		clilog.TeeError(cmd.ErrOrStderr(), err.Error())
		return
	}

	// TODO allow user to provide row count via --head to set last
	results, err := connection.Client.GetTextResults(s, 0, 500)
	if err != nil {
		clilog.TeeError(cmd.ErrOrStderr(), err.Error())
		return
	}

	if outfile, err := cmd.Flags().GetString("output"); err != nil {
		clilog.TeeError(cmd.ErrOrStderr(), err.Error())
		return
	} else if outfile != "" {
		f, err := os.Create(outfile)
		if err != nil {
			clilog.TeeError(cmd.ErrOrStderr(), err.Error())
			return
		}
		defer f.Close()
		f.WriteString(fmt.Sprintf("%v", results))
	} else {
		for _, e := range results.Entries {
			fmt.Printf("%s\n", e.Data)
		}
		//fmt.Printf("%#v\n", results)
	}
}

// Pulls a query from args or a reference uuid, depending on if the latter is given
func FetchQueryString(cmd *cobra.Command, args []string) (query string, err error) {
	var ref string // query library uuid
	if ref, err = cmd.Flags().GetString("reference"); err != nil {
		return "", err
	} else if strings.TrimSpace(ref) != "" {
		if err := uuid.Validate(ref); err != nil {
			return "", err
		}
		uuid, err := uuid.Parse(ref)
		if err != nil {
			return "", err
		}
		sl, err := connection.Client.GetSearchLibrary(uuid)
		if err != nil {
			return "", err
		}
		return sl.Query, nil
	}

	query = strings.TrimSpace(strings.Join(args, " "))
	if query == "" { // superfluous query, don't bother
		return "", errors.New(ErrSuperfluousQuery)
	}
	return
}

//#endregion

// Validates and (if valid) submits the given query to the connected server instance
func tryQuery(qry string, duration time.Duration) (grav.Search, error) {
	var err error
	// validate search query
	if err = connection.Client.ParseSearch(qry); err != nil {
		return grav.Search{}, fmt.Errorf("'%s' is not a valid query: %s", qry, err.Error())
	}

	start := time.Now()
	sreq := types.StartSearchRequest{
		SearchStart:  start.Format(timeFormat),
		SearchEnd:    start.Add(duration).Format(timeFormat),
		Background:   false,
		SearchString: qry, // pull query from the commandline
	}
	go func() {
		clilog.Writer.Infof("Executing foreground search '%v' from %v -> %v",
			sreq.SearchString, sreq.SearchStart, sreq.SearchEnd)
	}()
	return connection.Client.StartSearchEx(sreq)
}

//#region actor implementation

type mode int8

const (
	inactive  mode = iota // prepared, but not utilized
	prompting             // accepting user input
	quitting              // leaving prompt
	waiting               // search submitted; waiting for results
)

type query struct {
	mode  mode
	error string

	curSearch  *grav.Search // nil or ongoing/recently-completed search
	searchDone chan string  // notification we can stop waiting

	help struct {
		model help.Model
		keys  helpKeyMap
	}
	ta textarea.Model
}

var Query action.Model = Initial()

func Initial() *query {
	q := &query{
		mode:       inactive,
		searchDone: make(chan string),
	}

	// configure text area
	q.ta = textarea.New()
	q.ta.ShowLineNumbers = true
	q.ta.Prompt = "->"
	q.ta.SetWidth(70)
	q.ta.SetHeight(5)

	// set up help
	q.help.model = help.New()
	q.help.keys = helpKeyMap{
		Submit: key.NewBinding(
			key.WithKeys("ctrl+enter"),
			key.WithHelp("ctrl+enter", "submit query"),
		),
		Quit: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "return to navigation"),
		),
	}

	return q
}

func (q *query) Update(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	switch q.mode {
	case quitting:
		return nil
	case waiting: // display spinner and wait
		// TODO
	case inactive:
		clilog.Writer.Debugf("Activating query model...")
		q.mode = prompting
		cmds = append(cmds, q.ta.Focus(), textarea.Blink)
		return tea.Batch(cmds...)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		clilog.Writer.Debugf("Recv'd key msg %v", msg)
		if key.Matches(msg, q.help.keys.Submit) {
			qry := q.ta.Value()
			if qry == "" {
				// superfluous request
				q.error = "empty request"
				// falls through to standard update
			} else {
				clilog.Writer.Infof("Submitting query '%v'...", qry)
				// TODO take duration from second viewport
				var duration time.Duration = 1 * time.Hour
				s, err := tryQuery(qry, duration)
				if err != nil {
					q.error = err.Error()
					return tea.Batch()
				}
				q.curSearch = &s

				q.mode = waiting
				return tea.Batch(cmds...)
			}
		}
	}

	var cmd tea.Cmd
	q.ta, cmd = q.ta.Update(msg)
	return cmd
}

func (q *query) View() string {

	ch := make(chan string)
	go func() {
		ch <- q.ta.View()
		close(ch)
	}() // TODO sometimes ta.View gets hard stuck
	h := q.help.model.View(q.help.keys)

	ta := <-ch

	return fmt.Sprintf("Query:\n%s\n%s", ta, h)
}

func (q *query) Done() bool {
	return q.mode == quitting
}

func (q *query) Reset() error {
	q.mode = inactive
	q.error = ""
	q.curSearch = nil
	q.ta.Reset()
	return nil
}

func (q *query) SetArgs(_ *pflag.FlagSet, _ []string) (bool, error) {

	return true, nil
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

//#endregion
