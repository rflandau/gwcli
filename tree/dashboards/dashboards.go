package dashboards

import (
	"gwcli/action"
	"gwcli/tree/dashboards/delete"
	"gwcli/tree/dashboards/list"
	"gwcli/utilities/treeutils"

	"github.com/spf13/cobra"
)

const (
	use   string = "dashboards"
	short string = "list and manipulate dashboards"
	long  string = "list, edit (NYI), and delete dashboards."
)

var aliases []string = []string{"dashboard", "dash"}

func NewExtractorsNav() *cobra.Command {
	return treeutils.GenerateNav(use, short, long, aliases,
		[]*cobra.Command{},
		[]action.Pair{
			list.NewDashboardsListAction(),
			delete.NewDashboardDeleteAction(),
		})
}
