package actions

import (
	"fmt"
	"gwcli/treeutils"

	"github.com/spf13/cobra"
)

func NewKitsCmd() *cobra.Command {
	//u := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFAAAA")).Italic(true)

	cmd := &cobra.Command{
		Use:     treeutils.ActionStyle.Render("kits"),
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
