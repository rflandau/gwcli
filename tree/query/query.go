package query

import (
	"gwcli/action"
	"gwcli/clilog"
	"gwcli/treeutils"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
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

	return treeutils.GenerateAction(cmd, Query)
}


func initialLocalFlagSet() pflag.FlagSet {
	fs := pflag.FlagSet{}

	fs.DurationP("scheduled", "s", time.Second*30, "schedule this search to be run at a later date, over the given duration")

	// TODO
	//fs.StringP("name", "n", "", "the shorthand that will be expanded")
	//fs.StringP("description", "d", "", "(flavour) description")
	//fs.StringP("expansion", "e", "", "value for the macro to expand to")

	return fs
}

//#endregion

//#region cobra command

func run(cmd *cobra.Command, args []string) {
	// fetch query from cli
	clilog.Writer.Debugf("Passed arguments found: %v", args)
}

	


//#endregion

//#region actor implementation

type query struct {
	done bool
	ta textarea.Model
}

var Query action.Model

//#endregion