package query

import (
	"fmt"
	"gwcli/action"
	"gwcli/clilog"
	"gwcli/connection"
	"gwcli/treeutils"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/gravwell/gravwell/v3/client/types"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var localFS pflag.FlagSet

//#region command/action set up

func GenerateAction() action.Pair {
	cmd := treeutils.NewActionCommand("query", "submit a query",
		"Generate and send a query to the remote server. Results can be received via this cli or later on the web GUI.\n"+
		"All arguments after `query` will be passed to the instance as the search command.", []string{}, run)

	localFS = initialLocalFlagSet()

	cmd.Flags().AddFlagSet(&localFS)

	cmd.MarkFlagsOneRequired("duration")

	cmd.MarkFlagsRequiredTogether("name", "description", "schedule")

	return treeutils.GenerateAction(cmd, Query)
}


func initialLocalFlagSet() pflag.FlagSet {
	fs := pflag.FlagSet{}

	fs.DurationP("duration", "t", time.Hour*1, "the amount of time over which the query should be run.")
	fs.StringP("reference", "r", "", "a reference to a query library item to execute instead of a provided query.")

	// scheduled searches
	fs.StringP("name", "n", "", "name for a scheduled search")
	fs.StringP("description", "d", "", "(flavour) description")
	fs.StringP("schedule", "s", "", "schedule this search to be run at a later date, over the given duration.")

	return fs
}

//#endregion

//#region cobra command

func run(cmd *cobra.Command, args []string) {
	// fetch query from cli
	clilog.Writer.Debugf("Passed arguments found: %v", args)

	if schedule, err := cmd.Flags().GetString("schedule"); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err); return
	} else if schedule != ""{
		// TODO implement scheduled searches
	}

	// fetch required flags
	duration, err := cmd.Flags().GetDuration("duration")
	if err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err); return
	}
	
	// parse query from args or use reference (if given)


	start := time.Now()

	sreq := types.StartSearchRequest{SearchStart: start.String(), SearchEnd: start.Add(duration).String()}
	connection.Client.StartSearchEx(sreq)
}

	


//#endregion

//#region actor implementation

type query struct {
	done bool
	ta textarea.Model
}

var Query action.Model

// ParseSearch validator

//#endregion