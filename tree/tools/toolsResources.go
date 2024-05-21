package tools

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tools",
		Short:   "Tools & Resources submenu",
		Long:    `Actions associated to tooling and assets/resources`,
		Aliases: []string{"resources"},
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("tools called") // TODO
		},
	}

	// associate subcommands

	return cmd
}
