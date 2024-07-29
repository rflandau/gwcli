package kits

import (
	"gwcli/action"
	"gwcli/tree/kits/list"
	"gwcli/utilities/treeutils"

	"github.com/spf13/cobra"
)

const (
	use   string = "kits"
	short string = "List and manipulate kits"
	long  string = "..."
)

var aliases []string = []string{"kit"}

func NewKitsNav() *cobra.Command {
	return treeutils.GenerateNav(use, short, long, aliases,
		[]*cobra.Command{},
		[]action.Pair{list.NewKitsListAction()})
}
