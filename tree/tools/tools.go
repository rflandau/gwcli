package tools

import (
	"gwcli/tree/tools/macros"
	"gwcli/treeutils"

	"github.com/spf13/cobra"
)

var (
	use     string   = "tools"
	short   string   = "Tools & Resources submenu"
	long    string   = "Actions associated to tooling and assets/resources"
	aliases []string = []string{"resources"}
)

func GenerateTree() *cobra.Command {
	return treeutils.GenerateNav(use, short, long, aliases, []*cobra.Command{macros.GenerateNav()}, nil)
}
