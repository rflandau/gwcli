package macros

import (
	"gwcli/action"
	"gwcli/tree/macros/create"
	"gwcli/tree/macros/delete"
	"gwcli/tree/macros/edit"
	"gwcli/tree/macros/list"
	"gwcli/utilities/treeutils"

	"github.com/spf13/cobra"
)

const (
	use   string = "macros"
	short string = "create, edit, delete, and view macros"
	long  string = "Macros are search keywords that expand to preset phrases on use within a query."
)

var aliases []string = []string{"macro", "m"}

func NewMacrosNav() *cobra.Command {
	return treeutils.GenerateNav(use, short, long, aliases, []*cobra.Command{},
		[]action.Pair{list.NewMacroListAction(),
			create.NewMacroCreateAction(),
			delete.NewMacroDeleteAction(),
			edit.NewMacroEditAction()})
}
