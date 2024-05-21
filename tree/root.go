/**
 * Root node of the command tree and the true "main".
 * Initializes itself and `Executes()`, triggering Cobra to assemble itself.
 */
package tree

import (
	"gwcli/tree/search"
	"gwcli/tree/systems"
	"gwcli/tree/tools"
	"gwcli/treeutils"
	"os"

	"github.com/spf13/cobra"
)

func EnforceLogin(cmd *cobra.Command, args []string) error {
	// TODO check for token or supply user with login model
	return nil
}

/** Generate Flags populates all root-relevant flags (ergo global and root-local flags) */
func GenerateFlags(root *cobra.Command) {
	root.PersistentFlags().Bool("no-interactive", false, "Disallows gwcli from entering interactive mode and prints context help instead.\nRecommended for use in scripts to avoid hanging on a malformed command")
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	var rootCmd = &cobra.Command{
		Use:   "gwcli",
		Short: "Gravwell CLI Client",
		Long: "gwcli is a CLI client for interacting with your Gravwell instance directly from your terminal.\n" +
			"It can be used non-interactively in your scripts or interactively via the built-in TUI.\n" +
			"To invoke the TUI, simply call `gwcli`.",
		PersistentPreRunE: EnforceLogin,
		Run:               treeutils.NavRun,
	}

	// associate flags
	GenerateFlags(rootCmd)

	// set up nav and action groups
	rootCmd.AddGroup(&cobra.Group{ID: treeutils.NavID, Title: "Navigation"})
	rootCmd.AddGroup(&cobra.Group{ID: treeutils.ActionID, Title: "Actions"})

	// add root to nav group
	rootCmd.GroupID = treeutils.NavID

	// add direct descendents, which will each add their descendents
	rootCmd.AddCommand(systems.GenerateTree())
	rootCmd.AddCommand(search.GenerateTree())
	rootCmd.AddCommand(tools.GenerateTree())

	if !rootCmd.AllChildCommandsHaveGroup() {
		// TODO move this into a testing package
		panic("some children missing a group")
	}

	// configure Windows mouse trap
	cobra.MousetrapHelpText = "This is a command line tool.\n" +
		"You need to open cmd.exe and run it from there.\n" +
		"Press Return to close.\n"
	cobra.MousetrapDisplayDuration = 0

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
