package resources

import (
	"gwcli/action"
	"gwcli/tree/resources/list"
	"gwcli/utilities/treeutils"

	"github.com/spf13/cobra"
)

const (
	use   string = "resources"
	short string = "system resources submenu"
	long  string = "Create, list, edit (NYI), and delete resources."
)

var aliases []string = []string{"resources"}

func NewResourcesNav() *cobra.Command {
	return treeutils.GenerateNav(use, short, long, aliases,
		[]*cobra.Command{},
		[]action.Pair{list.NewResourcesListAction()})
}
