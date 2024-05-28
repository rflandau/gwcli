package kits

import (
	"gwcli/action"
	"gwcli/tree/kits/kitactions"
	"gwcli/treeutils"

	"github.com/spf13/cobra"
)

var (
	use     string   = "kits"
	short   string   = "List and manipulate kits"
	long    string   = "..."
	aliases []string = []string{"kit"}
)

func NewKitsNav() *cobra.Command {
	// no sub navs
	return treeutils.GenerateNav(use, short, long, aliases, []*cobra.Command{}, []action.Pair{kitactions.NewListCmd()})
}
