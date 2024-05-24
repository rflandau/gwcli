package search

import (
	"gwcli/tree/search/kits"
	"gwcli/treeutils"

	"github.com/spf13/cobra"
)

var (
	use     string   = "search"
	short   string   = "Search & Data submenu"
	long    string   = "Actions associated to performing, previewing searches and managing, manipulating data"
	aliases []string = []string{"data", "health"}
)

func NewSearchCmd() *cobra.Command {
	return treeutils.GenerateNav(use, short, long, aliases, []*cobra.Command{kits.NewKitsNav()}, nil)
}
