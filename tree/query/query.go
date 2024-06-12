package query

import (
	"fmt"
	"gwcli/action"
	"gwcli/clilog"
	cobraspinner "gwcli/cobra_spinner"
	"gwcli/connection"
	"gwcli/treeutils"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/gravwell/gravwell/v3/client/types"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// the Gravwell client can only consume time formatted as follows
const timeFormat = "2006-01-02T15:04:05.999999999Z07:00"

var localFS pflag.FlagSet

//#region command/action set up

func GenerateAction() action.Pair {
	cmd := treeutils.NewActionCommand("query", "submit a query",
		"Generate and send a query to the remote server. Results can be received via this cli or later on the web GUI.\n"+
			"All arguments after `query` will be passed to the instance as the string to query.", []string{}, run)

	localFS = initialLocalFlagSet()

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

	q, err := GenerateQueryString(cmd, args)
	if err != nil {
		clilog.TeeError(cmd.ErrOrStderr(), err.Error())
		return
	}

	start := time.Now()
	sreq := types.StartSearchRequest{
		SearchStart:  start.Format(timeFormat),
		SearchEnd:    start.Add(duration).Format(timeFormat),
		Background:   false,
		SearchString: q, // pull query from the commandline
	}
	go func() {
		clilog.Writer.Infof("Executing foreground search '%v' from %v -> %v",
			sreq.SearchString, sreq.SearchStart, sreq.SearchEnd)
	}()
	s, err := connection.Client.StartSearchEx(sreq)
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
func GenerateQueryString(cmd *cobra.Command, args []string) (query string, err error) {
	var ref string // query library uuid
	if ref, err = cmd.Flags().GetString("reference"); err != nil {
		return "", err
	} else if strings.TrimSpace(ref) != "" {
		clilog.Writer.Infof("Search ref uuid '%v'", ref)
		// TODO look up query by ref, if given
		// return query, nil
	}

	query = strings.Join(args, " ")
	// validate search query
	if err = connection.Client.ParseSearch(query); err != nil {
		query = ""
		err = fmt.Errorf("'%s' is not a valid query: %s", query, err.Error())
	}
	return
}

//#endregion

//#region actor implementation

type query struct {
	done bool
	ta   textarea.Model
}

var Query action.Model

// ParseSearch validator

//#endregion
