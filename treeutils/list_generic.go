/**
 * Helper functions and generic struct.
 * Intended to be boilder plate for specific list implementations.
 */

package treeutils

import (
	"fmt"
	"gwcli/clilog"
	"gwcli/connection"
	"gwcli/weave"
	"reflect"

	tea "github.com/charmbracelet/bubbletea"
	grav "github.com/gravwell/gravwell/v3/client"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// NewListCmd creates and returns a cobra.Command suitable for use as a list
// action, complete with common flags and a generic run function operating off
// the given dataFunc.
//
// Flags: {--csv, --json, --table} --columns <...>
//
// If no output module is given, defaults to --table.
//
// ! `dataFunc` should be a static wrapper function for a method that returns an array of structures containing the data to be listed.
// ! `dataStruct` must be the type of struct returned by dataFunc. Its values do not matter.
// Any data massaging required to get the data into an array of functions should be performed there.
// See kitactions' ListKits() as an example
//
// Go's Generics are a godsend.
func NewListCmd[Any any](use, short, long string, aliases []string, defaultColumns []string, dataStruct Any, dataFunc func(*grav.Client) ([]Any, error)) (*cobra.Command, ListAction) {
	// assert developer provided a usable data struct
	if reflect.TypeOf(dataStruct).Kind() != reflect.Struct {
		panic("dataStruct must be a struct") // developer error
	}

	// the function to run if called from the shell/non-interactively
	runFunc := func(cmd *cobra.Command, _ []string) {
		// check for --show-columns
		if sc, err := cmd.Flags().GetBool("show-columns"); err != nil {
			panic(err)
		} else if sc {
			col, err := weave.StructFields(dataStruct, true)
			if err != nil {
				panic(err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%+v\n", col)
			return
		}

		data, err := dataFunc(connection.Client)
		if err != nil {
			clilog.TeeError(cmd.ErrOrStderr(), err.Error())
			return
		}

		// process flags
		// NOTE format flags are marked mutually exclusive on creation
		//		we do not need to check for exclusivity here

		// determine columns
		var columns []string
		columns, err = cmd.Flags().GetStringSlice("columns")
		if err != nil {
			clilog.TeeError(cmd.ErrOrStderr(), err.Error())
			return
		}

		var format outputFormat = determineFormat(cmd)
		clilog.Writer.Debugf("List: format %s | row count: %d", format, len(data))
		switch format {
		case csv:
			fmt.Println(weave.ToCSV(data, columns))
		case json:
			//fmt.Println(weave.ToJSON(data, columns))
		case table:
			fmt.Println(weave.ToTable(data, columns))
		default:
			clilog.TeeError(cmd.ErrOrStderr(), fmt.Sprintf("unknown output format (%d)", format))
			return
		}
	}

	// generate the command
	cmd := NewActionCommand(use, short, long, aliases, runFunc)

	// define cmd-specific flag option
	fs := NewListFlagSet()
	cmd.Flags().AddFlagSet(&fs)
	cmd.MarkFlagsMutuallyExclusive("csv", "json", "table")

	// spin up a list action for interactive use
	la := NewListAction(defaultColumns)

	// share the flagset with the interactive action model

	return cmd, la
}

// Helper function for NewListCmd's runFunc creation
// Takes an initialized list cmd and returns the output format for listing
func determineFormat(cmd *cobra.Command) outputFormat {
	var format outputFormat
	if format_csv, err := cmd.Flags().GetBool("csv"); err != nil {
		panic(err)
	} else if format_csv {
		format = csv
	} else {
		if format_json, err := cmd.Flags().GetBool("json"); err != nil {
			panic(err)
		} else if format_json {
			format = json
		} else {

			format = table
		}
	}
	return format
}

func NewListFlagSet() pflag.FlagSet {
	fs := pflag.FlagSet{}
	fs.Bool("csv", false, "output results as csv")
	fs.Bool("json", false, "output results as json")
	fs.Bool("table", true, "output results in a human-readable table") // default
	fs.StringSlice("columns", []string{},
		"comma-seperated list of columns to include in the output."+
			"Use --show-columns to see the full list of columns.")
	fs.Bool("show-columns", false, "display the list of fully qualified column names and die.")

	return fs
}

//#region interactive mode (model) implementation

type ListAction struct {
	// data cleared by .Reset()
	done    bool
	format  outputFormat
	columns []string
	fs      pflag.FlagSet // current flagset, parsed or unparsed

	// data shielded from .Reset()
	DefaultFormat      outputFormat
	DefaultColumns     []string             // columns to output if unspecified
	DefaultFlagSetFunc func() pflag.FlagSet // flagset generation function used for .Reset()
}

// Constructs a ListAction suitable for interactive use
func NewListAction(defaultColumns []string) ListAction {
	fs := NewListFlagSet()
	return ListAction{fs: fs,
		DefaultFormat:      table,
		DefaultColumns:     defaultColumns,
		DefaultFlagSetFunc: NewListFlagSet}
}

func (la *ListAction) Update(msg tea.Msg) tea.Cmd {
	// TODO
	return nil
}

func (la *ListAction) View() string {
	return ""
}

func (la *ListAction) Done() bool {
	return la.done
}

func (la *ListAction) Reset() error {
	la.done = false
	la.format = la.DefaultFormat
	la.columns = la.DefaultColumns
	la.fs = la.DefaultFlagSetFunc()
	return nil
}

func (ls *ListAction) SetArgs(tokens []string) (bool, error) {
	// TODO
	return true, nil
}

//#endregion interactive mode (model) implementation
