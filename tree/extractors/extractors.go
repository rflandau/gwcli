package extractors

import (
	"gwcli/action"
	"gwcli/tree/extractors/create"
	"gwcli/tree/extractors/list"
	"gwcli/treeutils"

	"github.com/spf13/cobra"
)

var (
	use     string   = "extractors"
	short   string   = "List and manipulate extractors"
	long    string   = "Create, list, edit (NYI), and delete extractors."
	aliases []string = []string{"extractor", "ex", "ax", "autoextractor", "autoextractors"}
)

func NewExtractorsNav() *cobra.Command {
	return treeutils.GenerateNav(use, short, long, aliases,
		[]*cobra.Command{},
		[]action.Pair{
			list.NewExtractorsListAction(),
			create.NewExtractorsCreateAction()})
}
