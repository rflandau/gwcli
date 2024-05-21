package actions

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewHardwareCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "hardware",
		Short:   "Display information about the hardware underlying the instance",
		Long:    "...",
		Aliases: []string{"hw"},
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("hardware called") // TODO
		},
	}

	// associate subcommands

	return cmd
}
