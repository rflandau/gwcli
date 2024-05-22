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
	"time"

	"github.com/spf13/cobra"
)

func EnforceLogin(cmd *cobra.Command, args []string) error {
	// TODO check for token or supply user with login model
	return nil
}

/** Generate Flags populates all root-relevant flags (ergo global and root-local flags) */
func GenerateFlags(root *cobra.Command) {
	// global flags
	root.PersistentFlags().Bool("no-interactive", false, "Disallows gwcli from entering interactive mode and prints context help instead.\nRecommended for use in scripts to avoid hanging on a malformed command")
	root.PersistentFlags().StringP("username", "u", "", "login credential")
	root.PersistentFlags().StringP("password", "p", "", "login credential")
	root.MarkFlagsRequiredTogether("username", "password") // tie username+password together
	root.PersistentFlags().Bool("no-color", false, "Disables colourized output")
}

const ( // usage
	use   string = "gwcli"
	short string = "Gravwell CLI Client"
	long  string = "gwcli is a CLI client for interacting with your Gravwell instance directly from your terminal.\n" +
		"It can be used non-interactively in your scripts or interactively via the built-in TUI.\n" +
		"To invoke the TUI, simply call `gwcli`."
)

const ( // mousetrap
	mousetrapText string = "This is a command line tool.\n" +
		"You need to open cmd.exe and run it from there.\n" +
		"Press Return to close.\n"
	mousetrapDuration time.Duration = (0 * time.Second)
)

/**
 * Execute adds all child commands to the root command, sets flags
 * appropriately, and launches the program according to the given parameters
 * (via cobra.Command.Execute()).
 */
func Execute() {
	rootCmd := treeutils.GenerateNav(use, short, long, []string{}, systems.GenerateTree(), search.GenerateTree(), tools.GenerateTree())
	rootCmd.PersistentPreRunE = EnforceLogin
	rootCmd.Version = "prototype"

	// associate flags
	GenerateFlags(rootCmd)

	// TODO move this into a testing package
	if !rootCmd.AllChildCommandsHaveGroup() {
		panic("some children missing a group")
	}

	// configure Windows mouse trap
	cobra.MousetrapHelpText = mousetrapText
	cobra.MousetrapDisplayDuration = mousetrapDuration

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
