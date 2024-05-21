package systems

import (
	"fmt"
	"gwcli/tree/systems/actions"

	"github.com/spf13/cobra"
)

func GenerateTree() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "systems",
		Short:   "Systems & Health submenu",
		Long:    `Actions associated to monitoring the health and status of the system rit large`,
		Aliases: []string{"system", "health"},
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("systems called") // TODO
		},
	}

	// associate subcommands
	cmd.AddCommand(actions.NewHardwareCmd())

	return cmd
}
