/**
 * Contains utilities related to interacting with existing or former queries.
 * All query creation is done at the top-level query action.
 */
package queries

import (
	"gwcli/action"
	"gwcli/tree/tools/queries/scheduled"
	"gwcli/treeutils"

	"github.com/spf13/cobra"
)

var (
	use   string = "queries"
	short string = "List, delete, and manage existing and past queries"
	long  string = "Queries contians utilities for managing auxillary query actions." +
		"Query creation is handled by the top-level `query` action."
	aliases []string = []string{"searches"}
)

func NewQueriesNav() *cobra.Command {
	return treeutils.GenerateNav(use, short, long, aliases,
		[]*cobra.Command{scheduled.NewScheduledNav()},
		[]action.Pair{})
}
