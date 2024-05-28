/**
 * Root node of the command tree and the true "main".
 * Initializes itself and `Executes()`, triggering Cobra to assemble itself.
 * All invocations of the program operate via root, whether or not it hands off
 * control to Mother.
 */
package tree

import (
	"fmt"
	"gwcli/connection"
	"gwcli/tree/kits"
	"gwcli/tree/search"
	"gwcli/tree/systems"
	"gwcli/tree/tools"
	"gwcli/treeutils"
	"os"
	"time"

	"github.com/spf13/cobra"
)

/**
 * Logs the client into the Gravwell instance dictated by the --server flag.
 * Safe (ineffectual) to call if already logged in.
 */
func EnforceLogin(cmd *cobra.Command, args []string) error {
	if connection.Client == nil { // if we just started, initialize connection
		server, err := cmd.Flags().GetString("server")
		if err != nil {
			return err
		}
		if err = connection.Initialize(server); err != nil {
			return err
		}
	}

	// if logged in, we are done
	if connection.Client.LoggedIn() {
		return nil
	}

	// attempt to login
	u, err := cmd.Flags().GetString("username")
	if err != nil {
		return err
	}
	p, err := cmd.Flags().GetString("password")
	if err != nil {
		return err
	}

	// prompt for username and/or password
	// TODO
	if u == "" || p == "" {
		return fmt.Errorf("username (-u) and password (-p) required")
	}

	if err = connection.Login(u, p); err != nil {
		return err
	}

	fmt.Println("Logged in successfully")
	// TODO check for token or supply user with login model if interactivity available
	return nil

}

// TODO add lipgloss tree printing to help

/** Generate Flags populates all root-relevant flags (ergo global and root-local flags) */
func GenerateFlags(root *cobra.Command) {
	// global flags
	root.PersistentFlags().Bool("no-interactive", false, "Disallows gwcli from entering interactive mode and prints context help instead.\nRecommended for use in scripts to avoid hanging on a malformed command")
	root.PersistentFlags().StringP("username", "u", "", "login credential")
	root.PersistentFlags().StringP("password", "p", "", "login credential")
	root.MarkFlagsRequiredTogether("username", "password")                       // tie username+password together
	root.PersistentFlags().Bool("no-color", false, "Disables colourized output") // TODO via lipgloss.NoColor
	root.PersistentFlags().StringP("server", "s", "localhost:80", "<host>:<port>\nDefault: 'localhost:80'")
	// TODO JSON global flag output
	// TODO make the logger terse by default
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
	rootCmd := treeutils.GenerateNav(use, short, long, []string{}, []*cobra.Command{systems.NewSystemsNav(), search.NewSearchCmd(), tools.GenerateTree(), kits.NewKitsNav()}, nil)
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

	// configure root's Run to launch Mother
	rootCmd.Run = treeutils.NavRun

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
