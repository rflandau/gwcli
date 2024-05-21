package actions

import (
	"fmt"
	"gwcli/treeutils"

	"github.com/spf13/cobra"
)

func NewDiskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "disks",
		Short:   "Display information about the disks underlying the instance",
		Long:    "...",
		Aliases: []string{"disk"},
		GroupID: treeutils.ActionID,
		//PreRun: ,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("action called") // TODO
		},
	}
	return cmd
}
