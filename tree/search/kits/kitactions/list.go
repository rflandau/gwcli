package kitactions

import (
	"fmt"
	"gwcli/connection"
	"gwcli/treeutils"

	"github.com/spf13/cobra"
)

func NewListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all installed and staged kits",
		Long:    "...",
		Aliases: []string{},
		GroupID: treeutils.ActionID,
		//PreRun: ,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(connection.Client.ListKits())
		},
	}
	return cmd
}
