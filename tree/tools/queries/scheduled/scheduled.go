package scheduled

import (
	"gwcli/action"
	"gwcli/tree/tools/queries/scheduled/list"
	"gwcli/treeutils"

	"github.com/spf13/cobra"
)

var (
	use     string   = "scheduled"
	short   string   = "Manage scheduled queries"
	long    string   = "Alter and view previously scheduled queries"
	aliases []string = []string{}
)

func NewScheduledNav() *cobra.Command {
	return treeutils.GenerateNav(use, short, long, aliases,
		[]*cobra.Command{},
		[]action.Pair{list.NewScheduledQueriesListAction()})
}
