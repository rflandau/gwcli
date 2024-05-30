/**
 * Helper functions and generic structs.
 * Intended to be boilder plate for specific list implementation.
 */

package treeutils

import (
	"fmt"
	"gwcli/weave"

	"github.com/spf13/cobra"
)

type ListAction struct {
	done bool
}

type format = uint

const (
	json format = iota
	csv
	table
)

// NewListCmd creates and returns a cobra.Command suitable for use as a list
// action. Has common flags (such as mutually exclusive output modules, columns,
// inclusive/exclusive column handling) and is designated as an Action Cmd.
// If no output module is given, it will default to table
//
// `dataFunc` must be a function that returns an array of structures containing
// the data to be listed.
//
// Go's Generics are a godsend.
func NewListCmd[T any](use, short, long string, aliases []string, dataFunc func() ([]T, error)) *cobra.Command {
	// the problem is the run function
	// it is mostly generic, but will have a unique "fetchData" function
	// this would be easy to manage, but we do not know any function signatures in advance
	// 	and therefore cannot allow a user to pass in a function
	// ! for the time being, allow dataFunc to take no params

	// the function to run if called from the shell/non-interactively
	runFunc := func(cmd *cobra.Command, _ []string) {
		data, err := dataFunc()

		// process flags
		// NOTE format flags are marked mutually exclusive on creation
		//		we do not need to check for exclusivity here

		// determine columns
		var columns []string
		// TODO

		// determine output
		var format format = determineFormat(cmd)
		// TODO

		if err != nil {
			panic(err)
		}

		switch format {
		case csv:
			fmt.Println(weave.ToCSV(data, columns))
		case json:
			//fmt.Println(weave.ToJSON(data, columns))
		case table:
			//fmt.Println(weave.ToTable(data, columns))
		default:
			panic(fmt.Sprintf("unknown output format (%d)", format))
		}
	}

	// generate the command
	cmd := NewActionCommand(use, short, long, aliases, runFunc)

	// define flags
	cmd.Flags().Bool("csv", false, "output results as csv")
	cmd.Flags().Bool("json", false, "output results as json")
	cmd.Flags().Bool("table", true, "output results in a human-readable table") // default
	cmd.MarkFlagsMutuallyExclusive("csv", "json", "table")
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
