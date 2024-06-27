/**
 * Root node of the command tree and the true "main".
 * Initializes itself and `Executes()`, triggering Cobra to assemble itself.
 * All invocations of the program operate via root, whether or not it hands off
 * control to Mother.
 */
package tree

import (
	"errors"
	"fmt"
	"gwcli/action"
	"gwcli/clilog"
	"gwcli/connection"
	"gwcli/tree/kits"
	"gwcli/tree/query"
	"gwcli/tree/tools"
	"gwcli/tree/user"
	"gwcli/treeutils"
	"gwcli/utilities/usage"
	"os"
	"path"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	tokenFileName = "token"
	cfgSubFolder  = "gwcli" // $config_folder + configSubFolder
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
		if err = connection.Initialize(server, !insecure, insecure); err != nil {
			return err
		}
	}

	// if logged in, we are done
	if connection.Client.LoggedIn() {
		return nil
	}

	// login is attempted via JWT token first
	// If any stage in the process fails, the error is logged and we fall back to checking -u and -p
	// flags and then prompting for input
	if err := LoginViaToken(); err != nil {
		// jwt token failure; log and move on
		clilog.Writer.Warnf("Failed to login via JWT token: %v", err)

		// fetch credentials from flags
		u, err := cmd.Flags().GetString("username")
		if err != nil {
			return err
		}
		p, err := cmd.Flags().GetString("password")
		if err != nil {
			return err
		} else if p == "" {
			// try the password file
			pf, err := cmd.Flags().GetString("passfile")
			if err != nil {
				return err
			}
			if pf != "" {
				b, err := os.ReadFile(pf)
				if err != nil {
					return fmt.Errorf("failed to read password from %v: %v", pf, err)
				}
				p = strings.TrimSpace(string(b))
			}
		}

		// fetch additional data before attempting logon, if necessary
		if u == "" || p == "" {
			// if script mode, do not prompt
			if script, err := cmd.Flags().GetBool("script"); err != nil {
				clilog.Writer.Fatal("developer error: script flag is undefined")
			} else if script {
				return fmt.Errorf("no valid token found.\n" +
					"Please login via username (-u) and password (-p)")
			}

			// prompt for credentials
			creds, err := CredPrompt(u, p)
			if err != nil {
				return err
			}
			// pull input results
			if creds, ok := creds.(cred); !ok {
				return err
			} else if creds.killed {
				return errors.New("you must authenticate to use gwcli")
			} else {
				u = creds.UserTI.Value()
				p = creds.PassTI.Value()
			}
		}

		if err = connection.Login(u, p); err != nil {
			return err
		}

		if err := CreateToken(); err != nil {
			clilog.Writer.Warnf(err.Error())
			// failing to create the token is not fatal
		}
	}

	clilog.Writer.Infof("Logged in successfully")

	return nil

}

// Attempts to login via JWT token in the user's config directory.
// Returns an error on failures. This error should be considered nonfatal and the user logged in via
// an alternative method instead.
func LoginViaToken() (err error) {
	var (
		cfgDir   string
		tknbytes []byte
	)
	// NOTE the reversal of standard error checking (`err == nil`)
	if cfgDir, err = os.UserConfigDir(); err == nil {
		if tknbytes, err = os.ReadFile(path.Join(cfgDir, cfgSubFolder, tokenFileName)); err == nil {
			if err = connection.Client.ImportLoginToken(string(tknbytes)); err == nil {
				if err = connection.Client.TestLogin(); err == nil {
					return nil
				}
			}
		}
	}
	return
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
	if cfgDir, err := os.UserConfigDir(); err != nil {
		return fmt.Errorf("failed to determine pwd: %v\n not writing token", err)
	} else {
		if err = os.MkdirAll(path.Join(cfgDir, cfgSubFolder), 0700); err != nil {
			// check for exists error
			clilog.Writer.Debugf("mkdir error: %v", err)
			pe := err.(*os.PathError)
			if pe.Err != os.ErrExist {
				return fmt.Errorf("failed to ensure existance of directory %v: %v",
					path.Join(cfgDir, cfgSubFolder), err)
			}

		}
		tokenPath = path.Join(cfgDir, cfgSubFolder, tokenFileName)
	}

	// write out the token
	fd, err := os.OpenFile(tokenPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create token: %v", err)
	}
	if _, err := fd.WriteString(token); err != nil {
		return fmt.Errorf("failed to write token: %v", err)
	}

	if err = fd.Close(); err != nil {
		return fmt.Errorf("failed to close token file: %v", err)
	}

	clilog.Writer.Infof("Created token file @ %v", tokenPath)
	return nil
}

func ppost(cmd *cobra.Command, args []string) error {
	return connection.End()
}

// TODO add lipgloss tree printing to help

// Generate Flags populates all root-relevant flags (ergo global and root-local flags)
func GenerateFlags(root *cobra.Command) {
	// global flags
	root.PersistentFlags().Bool("script", false,
		"disallows gwcli from entering interactive mode and prints context help instead.\n"+
			"Recommended for use in scripts to avoid hanging on a malformed command.")
	root.PersistentFlags().StringP("username", "u", "", "login credential.")
	root.PersistentFlags().String("password", "", "login credential.")
	root.PersistentFlags().StringP("passfile", "p", "", "the path to a file containing your password")
	root.PersistentFlags().Bool("no-color", false, "disables colourized output.") // TODO via lipgloss.NoColor
	root.PersistentFlags().String("server", "localhost:80", "<host>:<port> of instance to connect to.\n")
	root.PersistentFlags().StringP("log", "l", "./gwcli.log", "log location for developer logs.\n")
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
		"To invoke the TUI, simply call `gwcli`."
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
		},
		[]action.Pair{
			query.NewQueryAction(),
		})
	rootCmd.SilenceUsage = true
	rootCmd.PersistentPreRunE = ppre
	rootCmd.PersistentPostRunE = ppost
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
