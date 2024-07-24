/**
 * Root node of the command tree and the true "main".
 * Initializes itself and `Executes()`, triggering Cobra to assemble itself.
 * All invocations of the program operate via root, whether or not it hands off
 * control to Mother.
 */
package tree

import (
	"gwcli/action"
	"gwcli/clilog"
	"gwcli/connection"
	"gwcli/group"
	"gwcli/tree/dashboards"
	"gwcli/tree/extractors"
	"gwcli/tree/kits"
	"gwcli/tree/query"
	"gwcli/tree/tools"
	"gwcli/tree/tree"
	"gwcli/tree/user"
	"gwcli/treeutils"
	"gwcli/utilities/cfgdir"
	"gwcli/utilities/usage"
	"time"

	"github.com/spf13/cobra"
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

	// if this is a 'complete' request, do not enforce login
	if cmd.Name() == cobra.ShellCompRequestCmd || cmd.Name() == cobra.ShellCompNoDescRequestCmd {
		return nil
	}

	// if this is a 'help' action, do not enforce login
	if cmd.Name() == "help" {
		return nil
	}

	return EnforceLogin(cmd, args)
}

// Logs the client into the Gravwell instance dictated by the --server flag.
// Safe (ineffectual) to call if already logged in.
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
		if err = connection.Initialize(server, !insecure, insecure, ""); err != nil {
			return err
		}
	}

	// generate credentials
	var (
		err    error
		script bool
		cred   connection.Credentials
	)
	if script, err = cmd.Flags().GetBool("script"); err != nil {
		return err
	}
	if cred.Username, err = cmd.Flags().GetString("username"); err != nil {
		return err
	}
	if cred.Password, err = cmd.Flags().GetString("password"); err != nil {
		return err
	}
	if cred.PassfilePath, err = cmd.Flags().GetString("passfile"); err != nil {
		return err
	}

	if err := connection.Login(cred, script); err != nil {
		return err
	}

	clilog.Writer.Infof("Logged in successfully")

	return nil

}

func ppost(cmd *cobra.Command, args []string) error {
	return connection.End()
}

// Generate Flags populates all root-relevant flags (ergo global and root-local flags)
func GenerateFlags(root *cobra.Command) {
	// global flags
	root.PersistentFlags().Bool("script", false,
		"disallows gwcli from entering interactive mode and prints context help instead.\n"+
			"Recommended for use in scripts to avoid hanging on a malformed command.")
	root.PersistentFlags().StringP("username", "u", "", "login credential.")
	root.PersistentFlags().String("password", "", "login credential.")
	root.PersistentFlags().StringP("passfile", "p", "", "the path to a file containing your password")
	root.PersistentFlags().Bool("no-color", false, "disables colourized output.")
	root.PersistentFlags().String("server", "localhost:80", "<host>:<port> of instance to connect to.\n")
	root.PersistentFlags().StringP("log", "l", cfgdir.DefaultStdLogPath, "log location for developer logs.\n")
	root.PersistentFlags().String("loglevel", "DEBUG", "log level for developer logs (-l).\n"+
		"Possible values: 'OFF', 'DEBUG', 'INFO', 'WARN', 'ERROR', 'CRITICAL', 'FATAL'.\n")
	root.PersistentFlags().Bool("insecure", false, "do not use HTTPS and do not enforce certs.")
}

const ( // usage
	use   string = "gwcli"
	short string = "Gravwell CLI Client"
	long  string = "gwcli is a CLI client for interacting with your Gravwell instance directly" +
		"from your terminal.\n" +
		"It can be used non-interactively in your scripts or interactively via the built-in TUI.\n" +
		"To invoke the TUI, simply call `gwcli`.\n" +
		"You can view help for any submenu or action by providing help a path.\n" +
		"For instance, try: `gwcli help tools macros create` or `gwcli query -h`"
)

const ( // mousetrap
	mousetrapText string = "This is a command line tool.\n" +
		"You need to open cmd.exe and run it from there.\n" +
		"Press Return to close.\n"
	mousetrapDuration time.Duration = (0 * time.Second)
)

// Execute adds all child commands to the root command, sets flags appropriately, and launches the
// program according to the given parameters
// (via cobra.Command.Execute()).
func Execute(args []string) int {
	rootCmd := treeutils.GenerateNav(use, short, long, []string{},
		[]*cobra.Command{
			tools.NewToolsNav(),
			kits.NewKitsNav(),
			user.NewUserNav(),
			extractors.NewExtractorsNav(),
			dashboards.NewExtractorsNav(),
		},
		[]action.Pair{
			query.NewQueryAction(),
			tree.NewTreeAction(),
		})
	rootCmd.SilenceUsage = true
	rootCmd.PersistentPreRunE = ppre
	rootCmd.PersistentPostRunE = ppost
	rootCmd.Version = "prototype"

	// associate flags
	GenerateFlags(rootCmd)

	if !rootCmd.AllChildCommandsHaveGroup() {
		panic("some children missing a group")
	}

	// configuration the completion command as an action
	rootCmd.SetCompletionCommandGroupID(group.ActionID)

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
