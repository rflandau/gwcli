package macros

import (
	"gwcli/action"
	"gwcli/tree/tools/macros/create"
	"gwcli/tree/tools/macros/delete"
	"gwcli/tree/tools/macros/list"
	"gwcli/treeutils"

	"github.com/spf13/cobra"
)

var (
	use     string   = "macros"
	short   string   = "Macro management submenu"
	long    string   = "Create, delete, and manage macros"
	aliases []string = []string{"macro"}
)

func GenerateNav() *cobra.Command {
	return treeutils.GenerateNav(use, short, long, aliases, []*cobra.Command{},
		[]action.Pair{list.NewListCmd(), create.GenerateAction(), delete.NewMacroDeleteAction()})
}
