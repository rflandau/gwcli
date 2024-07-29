package scheduled

import (
	"gwcli/action"
	"gwcli/tree/tools/queries/scheduled/create"
	"gwcli/tree/tools/queries/scheduled/delete"
	"gwcli/tree/tools/queries/scheduled/edit"
	"gwcli/tree/tools/queries/scheduled/list"

	"gwcli/utilities/treeutils"

	"github.com/spf13/cobra"
)

const (
	use   string = "scheduled"
	short string = "Manage scheduled queries"
	long  string = "Alter and view previously scheduled queries"
)

func NewScheduledNav() *cobra.Command {
	return treeutils.GenerateNav(use, short, long, []string{},
		[]*cobra.Command{},
		[]action.Pair{
			create.NewQueriesScheduledCreateAction(),
			list.NewScheduledQueriesListAction(),
			delete.NewQueriesScheduledDeleteAction(),
			edit.NewQueriesScheduledEditAction(),
		})
}
