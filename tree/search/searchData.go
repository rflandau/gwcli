package search

import (
	"fmt"
	"gwcli/tree/search/actions"

	"github.com/spf13/cobra"
)

func NewSearchCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "search",
		Short:   "Search & Data submenu",
		Long:    `Actions associated to performing, previewing searches and managing, manipulating data`,
		Aliases: []string{"data"},
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("search called") // TODO
		},
	}

	// associate subcommands
	cmd.AddCommand(actions.NewKitsCmd())
	return cmd
}
