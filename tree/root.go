/**
 * Root node of the command tree and the true "main".
 * Initializes itself and `Executes()`, triggering Cobra to assemble itself.
 * All invocations of the program operate via root, whether or not it hands off
 * control to Mother.
 */
package tree

import (
	"fmt"
	"gwcli/action"
	"gwcli/clilog"
	"gwcli/connection"
	"gwcli/tree/kits"
	"gwcli/tree/query"
	"gwcli/tree/systems"
	"gwcli/tree/tools"
	"gwcli/treeutils"
	"gwcli/utilities/usage"
	"os"
	"path"
	"time"

	"github.com/spf13/cobra"
)

const (
	tokenFileName = "token"
	envPathVar    = "GWCLI_TOKEN_PATH" // env key that maps to token path value
)

// global PersistenPreRunE.
//
// Ensures the logger is set up and the user has logged into the gravwell instance,
// completeing these actions if either is false
func ppre(cmd *cobra.Command, args []string) error {
	// set up the logger, if it is not already initialized
	if clilog.Writer == nil {
		path, err := cmd.Flags().GetString("log")
		if err != nil {
			return err
		}
		lvl, err := cmd.Flags().GetString("loglevel")
		if err != nil {
			return err
		}
		clilog.Init(path, lvl)
	}

	return EnforceLogin(cmd, args)
}

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
		insecure, err := cmd.Flags().GetBool("insecure")
		if err != nil {
			return err
		}
		if err = connection.Initialize(server, !insecure, insecure); err != nil {
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

	// TODO check for token an existing token at the path stored within the env var
	if v, found := os.LookupEnv(envPathVar); !found {
		// prompt for username and/or password
		// TODO
		if u == "" || p == "" {
			return fmt.Errorf("username (-u) and password (-p) required")
		}
		if err = connection.Login(u, p); err != nil {
			return err
		}

		if err := CreateToken(); err != nil {
			clilog.Writer.Warnf(err.Error())
			// failing to create the token is not fatal
		}
	} else {
		// load token from a file
		b, err := os.ReadFile(v)
		if err != nil {
			return err
		}
		connection.Client.ImportLoginToken(string(b))
		if err := connection.Client.TestLogin(); err != nil {
			return err
		}
		// TODO on failure, prompt for user/pass instead of dying
	}

	clilog.Writer.Infof("Logged in successfully")

	return nil

}

// Creates a login token for future use.
// The token's path is saved to an environment variable to be looked up on future runs
func CreateToken() error {
	var (
		err       error
		token     string
		tokenPath string
	)
	if token, err = connection.Client.ExportLoginToken(); err != nil {
		return fmt.Errorf("failed to export login token: %v", err)
	}
	if pwd, err := os.Getwd(); err != nil {
		return fmt.Errorf("failed to determine pwd: %v\n not writing token", err)
	} else {
		tokenPath = path.Join(pwd, tokenFileName)
	}

	// write out the token
	// TODO may need to create it as 0200 and change it to 0400 after writing
	fd, err := os.OpenFile(tokenPath, os.O_CREATE|os.O_WRONLY, 0400)
	if err != nil {
		return fmt.Errorf("failed to create token @ %v: %v", tokenPath, err)
	}
	if _, err := fd.WriteString(token); err != nil {
		return fmt.Errorf("failed to write token @ %v: %v", tokenPath, err)
	}

	// save its path as an environment variable
	if err := os.Setenv(envPathVar, tokenPath); err != nil {
		return fmt.Errorf("failed to set environment variable '%v' -> '%v': %v", envPathVar, tokenPath, err)
	}

	clilog.Writer.Infof("Created cred token @ %v with associated env var %v", tokenPath, envPathVar)
	return nil
}

// TODO add lipgloss tree printing to help

/** Generate Flags populates all root-relevant flags (ergo global and root-local flags) */
func GenerateFlags(root *cobra.Command) {
	// global flags
	root.PersistentFlags().Bool("no-interactive", false,
		"disallows gwcli from entering interactive mode and prints context help instead.\n"+
			"Recommended for use in scripts to avoid hanging on a malformed command.")
	root.PersistentFlags().StringP("username", "u", "", "login credential.")
	root.PersistentFlags().StringP("password", "p", "", "login credential.")
	root.MarkFlagsRequiredTogether("username", "password")
	root.PersistentFlags().Bool("no-color", false, "disables colourized output.") // TODO via lipgloss.NoColor
	root.PersistentFlags().String("server", "localhost:80", "<host>:<port> of instance to connect to.\n")
	root.PersistentFlags().StringP("log", "l", "./gwcli.log", "log location for developer logs.\n")
	root.PersistentFlags().String("loglevel", "DEBUG", "log level for developer logs (-l).\n"+
		"Possible values: 'OFF', 'DEBUG', 'INFO', 'WARN', 'ERROR', 'CRITICAL', 'FATAL'.\n")
	root.PersistentFlags().Bool("insecure", false, "do not use HTTPS and do not enforce certs.")
	// TODO JSON global flag output
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
func Execute(args []string) int {
	rootCmd := treeutils.GenerateNav(use, short, long, []string{},
		[]*cobra.Command{
			systems.NewSystemsNav(),
			tools.GenerateTree(),
			kits.NewKitsNav(),
		},
		[]action.Pair{
			query.GenerateAction(),
		})
	rootCmd.SilenceUsage = true
	rootCmd.PersistentPreRunE = ppre
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

	// if args were given (ex: we are in testing mode)
	// use those instead of os.Args
	if args != nil {
		rootCmd.SetArgs(args)
	}

	rootCmd.SetUsageFunc(usage.Usage)

	err := rootCmd.Execute()
	if err != nil {
		return 1
	}

	return 0
}
