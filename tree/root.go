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
	"gwcli/group"
	"gwcli/stylesheet"
	"gwcli/tree/kits"
	"gwcli/tree/query"
	"gwcli/tree/search"
	"gwcli/tree/systems"
	"gwcli/tree/tools"
	"gwcli/treeutils"
	"strings"
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

	// prompt for username and/or password
	// TODO
	if u == "" || p == "" {
		return fmt.Errorf("username (-u) and password (-p) required")
	}

	if err = connection.Login(u, p); err != nil {
		return err
	}

	clilog.Writer.Infof("Logged in successfully")
	// TODO check for token or supply user with login model if interactivity available
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
			search.NewSearchCmd(),
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

	rootCmd.SetUsageFunc(Usage)

	err := rootCmd.Execute()
	if err != nil {
		return 1
	}

	return 0
}

func Usage(c *cobra.Command) error {
	const titleWidth = 9

	var bldr strings.Builder
	bldr.WriteString(stylesheet.Header1Style.Width(titleWidth).Render("Usage:") + c.CommandPath() + "-u USER -p PASS")

	if c.GroupID == group.NavID { // nav
		bldr.WriteString(" [subcommand]\n")
	} else { // action
		bldr.WriteString(" [flags]\n")
		bldr.WriteString(stylesheet.Header1Style.Render("Local Flags:") + "\n")
		bldr.WriteString(c.LocalNonPersistentFlags().FlagUsages() + "\n")
	}

	bldr.WriteString(stylesheet.Header1Style.Render("Global Flags:") + "\n")
	bldr.WriteString(c.Root().PersistentFlags().FlagUsages() + "\n")

	if len(c.Aliases) != 0 {
		var s strings.Builder
		s.WriteString(stylesheet.Header1Style.Width(titleWidth).Render("Aliases:") + " ")
		for _, a := range c.Aliases {
			s.WriteString(a + " ")
		}
		bldr.WriteString(strings.TrimSpace(s.String()) + "\n")
	}

	// split children by group
	navs := make([]*cobra.Command, 0)
	actions := make([]*cobra.Command, 0)
	children := c.Commands()
	for _, c := range children {
		if c.GroupID == group.NavID {
			navs = append(navs, c)
		} else {
			actions = append(actions, c)
		}
	}

	// output navs as submenus
	if len(navs) > 0 {
		var s strings.Builder
		s.WriteString("\n" + stylesheet.Header1Style.Render("Submenus"))
		for _, n := range navs {
			s.WriteString("\n  " + stylesheet.NavStyle.Render(n.Name()))
		}
		bldr.WriteString(s.String() + "\n")
	}

	// output actions
	if len(actions) > 0 {
		var s strings.Builder
		s.WriteString("\n" + stylesheet.Header1Style.Render("Actions"))
		for _, a := range actions {
			s.WriteString("\n  " + stylesheet.ActionStyle.Render(a.Name()))
		}
		bldr.WriteString(s.String() + "\n")
	}

	fmt.Fprintln(c.OutOrStdout(), bldr.String())
	return nil
}
