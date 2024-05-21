package tree

import (
	"fmt"

	"github.com/spf13/cobra"
)

// toolsCmd represents the toolsResources command
var toolsCmd = &cobra.Command{
	Use:     "tools",
	Short:   "Tools & Resources submenu",
	Long:    `Actions associated to tooling and assets/resources`,
	Aliases: []string{"resources"},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("tools called") // TODO
	},
}

func init() {
	rootCmd.AddCommand(toolsCmd)
}
