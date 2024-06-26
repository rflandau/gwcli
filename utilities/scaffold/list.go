// A list action runs a given function that outputs an arbitrary data structure.
// The results are sent to weave and packaged in a way that can be listed for the user.
// This provides a consistent interface for actions that list arbitrary data.
package scaffold

import (
	"fmt"
	"gwcli/action"
	"gwcli/clilog"
	"gwcli/connection"
	"gwcli/stylesheet"
	"gwcli/treeutils"
	"reflect"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	grav "github.com/gravwell/gravwell/v3/client"
	"github.com/gravwell/gravwell/v3/utils/weave"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

//#region enumeration

type outputFormat uint

const (
	json outputFormat = iota
	csv
	table
	unknown
)

func (f outputFormat) String() string {
	switch f {
	case json:
		return "JSON"
	case csv:
		return "CSV"
	case table:
		return "table"
	}
	return fmt.Sprintf("unknown format (%d)", f)
}

//#endregion enumeration

// Function that retrieves an array of structs of type dataStruct
type dataFunction[Any any] func(*grav.Client, *pflag.FlagSet) ([]Any, error)

const use = "list"

// NewListCmd creates and returns a cobra.Command suitable for use as a list
// action, complete with common flags and a generic run function operating off
// the given dataFunction.
//
// Flags: {--csv|--json|--table} [--columns ...]
//
// If no output module is given, defaults to --table.
//
// ! `dataFn` should be a static wrapper function for a method that returns an array of structures
// containing the data to be listed.
//
// ! `dataStruct` must be the type of struct returned in array by dataFunc.
// Its values do not matter.
//
// Any data massaging required to get the data into an array of structures should be performed in
// the data function. Non-list-standard flags (ex: those passed to addtlFlags, if not nil) should
// also be handled in the data function.
// See tree/kits/list's ListKits() as an example.
//
// Go's Generics are a godsend.
func NewListCmd[Any any](short, long string,
	aliases []string, defaultColumns []string,
	dataStruct Any, dataFn dataFunction[Any], addtlFlags *pflag.FlagSet) action.Pair {
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
			cols, err := weave.StructFields(dataStruct, true)
			if err != nil {
				panic(err)
			}
			fmt.Println(strings.Join(cols, " "))
			return
		}

		// fetch columns
		var (
			columns []string
			err     error
		)
		columns, err = cmd.Flags().GetStringSlice("columns")
		if err != nil {
			panic(err)
		}
		if len(columns) == 0 {
			columns = defaultColumns
		}

		// check for --no-color
		noColor, err := cmd.Flags().GetBool("no-color")
		if err != nil {
			panic(err)
		}

		output, err := List(cmd.Flags(), columns, !noColor, dataStruct, dataFn)
		if err != nil {
			clilog.TeeError(cmd.ErrOrStderr(), err.Error())
		}
		fmt.Println(output)
	}

	// generate the command
	cmd := treeutils.NewActionCommand(use, short, long, aliases, runFunc)

	// set up flags
	fs := pflag.FlagSet{}
	fs.Bool("csv", false, "output results as csv")
	fs.Bool("json", false, "output results as json")
	fs.Bool("table", true, "output results in a human-readable table") // default
	fs.StringSlice("columns", []string{},
		"comma-seperated list of columns to include in the output."+
			"Use --show-columns to see the full list of columns.")
	fs.Bool("show-columns", false, "display the list of fully qualified column names and die.")

	// attach normal list flags and, if applicable, additional flags
	cmd.Flags().AddFlagSet(&fs)
	if addtlFlags != nil {
		cmd.Flags().AddFlagSet(addtlFlags)
	}
	cmd.Flags().SortFlags = false // does not seem to be respected
	cmd.MarkFlagsMutuallyExclusive("csv", "json", "table")

	// spin up a list action for interactive use
	la := NewListAction(defaultColumns, dataStruct, dataFn, *cmd.Flags())

	return treeutils.GenerateAction(cmd, &la)
}

// Given a **parsed** flagset, determines and returns output format
func determineFormat(fs *pflag.FlagSet) outputFormat {
	if !fs.Parsed() {
		return unknown
	}
	var format outputFormat
	if format_csv, err := fs.GetBool("csv"); err != nil {
		panic(err)
	} else if format_csv {
		format = csv
	} else {
		if format_json, err := fs.GetBool("json"); err != nil {
			panic(err)
		} else if format_json {
			format = json
		} else {

			format = table
		}
	}
	return format
}

// Driver function to call the provided data func and format its output via weave
func List[Any any](fs *pflag.FlagSet, columns []string, color bool,
	dataStruct Any, dataFn dataFunction[Any]) (string, error) {

	data, err := dataFn(connection.Client, fs)
	if err != nil {
		return "", err
	}

	// NOTE format flags are marked mutually exclusive on creation
	//		we do not need to check for exclusivity here
	var format outputFormat = determineFormat(fs)
	clilog.Writer.Debugf("List: format %s | row count: %d", format, len(data))
	toRet, err := "", nil
	switch format {
	case csv:
		toRet = weave.ToCSV(data, columns)
	case json:
		toRet, err = weave.ToJSON(data, columns)
	case table:
		if color {
			toRet = weave.ToTable(data, columns, stylesheet.Table)
		} else {
			toRet = weave.ToTable(data, columns) // omit table styling

		}
	default:
		toRet = ""
		err = fmt.Errorf(fmt.Sprintf("unknown output format (%d)", format))
	}
	return toRet, err
}

//#region interactive mode (model) implementation

type ListAction[Any any] struct {
	// data cleared by .Reset()
	done        bool
	columns     []string
	showColumns bool          // print columns and exit
	fs          pflag.FlagSet // current flagset, parsed or unparsed

	// data shielded from .Reset()
	DefaultFormat  outputFormat
	DefaultColumns []string      // columns to output if unspecified
	baseFS         pflag.FlagSet // unparsed flagset to restore to
	color          bool          // inferred from the global "--no-color" flag

	// individualized for each user of list_generic
	dataStruct Any
	dataFunc   dataFunction[Any]
}

// Constructs a ListAction suitable for interactive use
func NewListAction[Any any](defaultColumns []string, dataStruct Any, dFn dataFunction[Any],
	baseFS pflag.FlagSet) ListAction[Any] {

	la := ListAction[Any]{
		columns:        defaultColumns,
		fs:             baseFS,
		DefaultFormat:  table,
		DefaultColumns: defaultColumns,
		baseFS:         baseFS,
		dataStruct:     dataStruct,
		dataFunc:       dFn}

	return la
}

func (la *ListAction[T]) Update(msg tea.Msg) tea.Cmd {
	if la.done {
		return nil
	}

	// check for --show-columns
	if la.showColumns {
		cols, err := weave.StructFields(la.dataStruct, true)
		if err != nil {
			panic(err)
		}
		return tea.Println(strings.Join(cols, " "))
	}

	s, err := List(&la.fs, la.columns, la.color, la.dataStruct, la.dataFunc)
	if err != nil {
		panic(err)
	}

	la.done = true

	return tea.Println(s)
}

func (la *ListAction[T]) View() string {
	return ""
}

// Called once per cycle to test if Mother should reassert control
func (la *ListAction[T]) Done() bool {
	return la.done
}

// Called when the action is unseated by Mother on exiting handoff mode
func (la *ListAction[T]) Reset() error {
	la.done = false
	la.columns = la.DefaultColumns
	la.showColumns = false
	la.fs = la.baseFS
	return nil
}

var _ action.Model = &ListAction[any]{}

// Called when the action is invoked by the user and Mother *enters* handoff mode
// Mother parses flags and provides us a handle to check against
func (la *ListAction[T]) SetArgs(
	inherited *pflag.FlagSet, tokens []string) (invalid string, onStart []tea.Cmd, err error) {

	err = la.fs.Parse(tokens)
	if err != nil {
		return "", nil, err
	}
	fs := la.fs

	// parse column handling
	// only need to parse columns if user did not pass in --show-columns
	if la.showColumns, err = fs.GetBool("show-columns"); err != nil {
		return "", nil, err
	} else if !la.showColumns {
		// fetch columns if it exists
		if cols, err := fs.GetStringSlice("columns"); err != nil {
			return "", nil, err
		} else if len(cols) > 0 {
			la.columns = cols
		} // else: defaults to DefaultColumns
	}

	nc, err := inherited.GetBool("no-color")
	if err != nil {
		la.color = false
		clilog.Writer.Warnf("Failed to fetch no-color from inherited: %v", err)
	}
	la.color = !nc

	return "", nil, nil
}

//#endregion interactive mode (model) implementation
