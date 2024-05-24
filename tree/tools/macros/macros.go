package macros

import (
	"gwcli/action"
	"gwcli/tree/tools/macros/macrosactions"
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
	return treeutils.GenerateNav(use, short, long, aliases, []*cobra.Command{}, []action.Pair{macrosactions.GenerateAction()})
}
