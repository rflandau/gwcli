package actions

import (
	"fmt"
	"gwcli/treeutils"

	"github.com/spf13/cobra"
)

func NewHardwareCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "hardware",
		Short:   "Display information about the hardware underlying the instance",
		Long:    "...",
		Aliases: []string{"hw"},
		GroupID: treeutils.ActionID,
		//PreRun: ,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("action called") // TODO
		},
	}
	return cmd
}
