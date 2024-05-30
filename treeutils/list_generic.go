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

	grav "github.com/gravwell/gravwell/v3/client"
	"github.com/spf13/cobra"
)

type ListAction struct {
	done bool
}

type format uint

const (
	json format = iota
	csv
	table
)

func (f format) String() string {
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

// NewListCmd creates and returns a cobra.Command suitable for use as a list
// action, complete with common flags and a generic run function operating off
// the given dataFunc.
//
// Flags: {--csv, --json, --table} --columns <...>
//
// If no output module is given, defaults to --table.
//
// ! `dataFunc` should be a static wrapper function for a method that returns an array of structures containing the data to be listed.
// Any data massaging required to get the data into an array of functions should be performed there.
// See kitactions' ListKits() as an example
//
// Go's Generics are a godsend.
func NewListCmd[Any any](use, short, long string, aliases []string, dataFunc func(*grav.Client) ([]Any, error)) *cobra.Command {
	// the function to run if called from the shell/non-interactively
	runFunc := func(cmd *cobra.Command, _ []string) {
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

		var format format = determineFormat(cmd)
		clilog.Writer.Debugf("List: format %s | row count: %d", format, len(data))
		switch format {
		case csv:
			fmt.Println(weave.ToCSV(data, columns))
		case json:
			//fmt.Println(weave.ToJSON(data, columns))
		case table:
			//fmt.Println(weave.ToTable(data, columns))
		default:
			clilog.TeeError(cmd.ErrOrStderr(), fmt.Sprintf("unknown output format (%d)", format))
			return
		}
	}

	// generate the command
	cmd := NewActionCommand(use, short, long, aliases, runFunc)

	// define flags
	cmd.Flags().Bool("csv", false, "output results as csv")
	cmd.Flags().Bool("json", false, "output results as json")
	cmd.Flags().Bool("table", true, "output results in a human-readable table") // default
	cmd.MarkFlagsMutuallyExclusive("csv", "json", "table")
	cmd.Flags().StringSlice("columns", []string{},
		"comma-seperated list of columns to include in the output."+
			"Use --help to see the full list of columns.")
	// TODO add a flag (or modify help) to output possible columns
	return cmd
}

func determineFormat(cmd *cobra.Command) format {
	var format format
	if format_csv, err := cmd.Flags().GetBool("csv"); err != nil {
		panic(err)
	} else if format_csv {
		format = csv
	} else {
		if format_json, err := cmd.Flags().GetBool("csv"); err != nil {
			panic(err)
		} else if format_json {
			format = json
		} else {

			format = table
		}
	}
	return format
}
