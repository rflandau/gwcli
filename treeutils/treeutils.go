package treeutils

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

/** Creates and returns a Nav (tree node) that can now be assigned subcommands*/
func GenerateNav(use, short, long string, aliases []string, subCmds ...*cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:     use,
		Short:   short,
		Long:    long,
		Aliases: aliases,
		GroupID: NavID,
		//PreRun: ,
		Run: NavRun,
	}

	// associate groups
	AddNavGroup(cmd)
	AddActionGroup(cmd)

	// associate subcommands
	for _, sub := range subCmds {
		cmd.AddCommand(sub)
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

//#region groups

type GroupID = string

const (
	ActionID GroupID = "action"
	NavID    GroupID = "nav"
)

func AddNavGroup(cmd *cobra.Command) {
	cmd.AddGroup(&cobra.Group{ID: NavID, Title: "Navigation"})
}
func AddActionGroup(cmd *cobra.Command) {
	cmd.AddGroup(&cobra.Group{ID: ActionID, Title: "Actions"})
}

//#endregion
