package tools

import (
	"gwcli/action"
	"gwcli/tree/tools/macros"
	"gwcli/tree/tools/queries"
	"gwcli/utilities/treeutils"

	"github.com/spf13/cobra"
)

const (
	use   string = "tools"
	short string = "Tools & Resources submenu"
	long  string = "Actions associated to tooling and assets/resources"
)

var aliases []string = []string{"resources"}

func NewToolsNav() *cobra.Command {
	return treeutils.GenerateNav(use, short, long, aliases,
		[]*cobra.Command{macros.NewMacrosNav(), queries.NewQueriesNav()},
		[]action.Pair{})
}
