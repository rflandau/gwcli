package kits

import (
	"gwcli/tree/search/kits/kitactions"
	"gwcli/treeutils"

	"github.com/spf13/cobra"
)

var (
	use     string   = "kits"
	short   string   = "List and manipulate kits"
	long    string   = "..."
	aliases []string = []string{"kit"}
)

func NewKitsCmd() *cobra.Command {
	return treeutils.GenerateNav(use, short, long, aliases, kitactions.NewListCmd())
}
