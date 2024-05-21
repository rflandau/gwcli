package actions

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewKitsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "kits",
		Short:   "List and manipulate kits",
		Long:    "...",
		Aliases: []string{"kit"},
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("kits called") // TODO
		},
	}

	// associate subcommands

	return cmd
}
