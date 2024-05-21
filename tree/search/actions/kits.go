package actions

import (
	"fmt"
	"gwcli/treeutils"

	"github.com/spf13/cobra"
)

func NewKitsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "kits",
		Short:   "List and manipulate kits",
		Long:    "...",
		Aliases: []string{"kit"},
		GroupID: treeutils.ActionID,
		//PreRun: ,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("action called") // TODO
		},
	}
	return cmd
}
