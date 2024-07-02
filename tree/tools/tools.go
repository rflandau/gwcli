package tools

import (
	"gwcli/action"
	"gwcli/tree/tools/macros"
	"gwcli/tree/tools/queries"
	"gwcli/treeutils"

	"github.com/spf13/cobra"
)

var (
	use     string   = "tools"
	short   string   = "Tools & Resources submenu"
	long    string   = "Actions associated to tooling and assets/resources"
	aliases []string = []string{"resources"}
)

func NewToolsNav() *cobra.Command {
	return treeutils.GenerateNav(use, short, long, aliases,
		[]*cobra.Command{macros.NewMacrosNav(), queries.NewQueriesNav()},
		[]action.Pair{})
}
