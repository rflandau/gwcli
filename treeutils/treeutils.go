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

/* Creates and returns a Nav (tree node) that can now be assigned subcommands*/
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

/** Creates and returns an Action (tree leaf) that can be called directly
 * non-interactively or via associated methods (actions.Pair) interactively
 */
func GenerateAction(cmd *cobra.Command, act action.Model) action.Pair {
	return action.Pair{Action: cmd, Model: act}
}

/* Returns a boilerplate action command that can be fed into GenerateAction */
func NewActionCommand(use, short, long string, aliases []string, runFunc func(*cobra.Command, []string)) *cobra.Command {
	return &cobra.Command{
		Use:     use,
		Short:   short,
		Long:    long,
		Aliases: aliases,
		GroupID: group.ActionID,
		//PreRun: ,
		Run: runFunc,
	}
}

//#region cobra run functions

/**
 * NavRun is the Run function for all Navs (nodes).
 * It checks for the --no-interactive flag and initializes Mother with the
 * command as her pwd if no-interactive is unset.
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
