/**
 * Treeutils provides global utility functions to enforce consistency and
 * facillitate shared references across the tree.
 */
package treeutils

import (
	"fmt"
	"gwcli/action"
	"gwcli/group"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

/** Creates and returns a Nav (tree node) that can now be assigned subcommands*/
func GenerateNav(use, short, long string, aliases []string, navCmds []*cobra.Command, actionCmds []action.Pair) *cobra.Command {
	cmd := &cobra.Command{
		Use:     use,
		Short:   short,
		Long:    long,
		Aliases: aliases,
		GroupID: group.NavID,
		//PreRun: ,
		Run: NavRun,
	}

	// associate groups
	group.AddNavGroup(cmd)
	group.AddActionGroup(cmd)

	// associate subcommands
	for _, sub := range navCmds {
		cmd.AddCommand(sub)
	}
	for _, sub := range actionCmds {
		cmd.AddCommand(sub.Action)
		// now that the commands have a parent, add their models to map
		action.AddModel(sub.Action, sub.Model)
	}

	return cmd
}

//#region cobra run functions

/**
 * NavRun is the Run function for all Navs (nodes).
 * It checks for the --no-interactive flag and only initializes Mother if unset.
 */
var NavRun = func(cmd *cobra.Command, args []string) {
	noInteractive, err := cmd.Flags().GetBool("no-interactive")
	if err != nil {
		panic(err)
	}
	if noInteractive {
		cmd.Help()
	} else {
		fmt.Printf("Initializing Mother... (NYI)\n") // TODO initialize Mother here
	}
}

//#endregion

//#region lipgloss styling

/**
 * NOTE: Per the Lipgloss documentation (https://github.com/charmbracelet/lipgloss?tab=readme-ov-file#faq),
 * it is intelligent enough to automatically adjust or disable color depending on the given environment.
 */
var (
	ActionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFAAAA")) //.Italic(true)
	NavStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#AAAAFF"))
)

//#endregion
