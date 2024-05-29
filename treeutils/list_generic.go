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

func NewListCmd(use, short, long string, aliases []string, dataFunc func() ([]interface{}, error)) *cobra.Command {
	// the problem is the run function
	// it is mostly generic, but will have a unique "fetchData" function
	// this would be easy to manage, but we do not know any function signatures in advance
	// 	and therefore cannot allow a user to pass in a function
	// ! for the time being, allow dataFunc to take no params
	// ? the interface array parameter in dataFunc may be unacceptable

	runFunc := func(cmd *cobra.Command, _ []string) {
		data, err := dataFunc()

		// determine columns from flags
		var columns []string
		// TODO

		if err != nil {
			panic(err)
		}
		if csv, err := cmd.Flags().GetBool("csv"); err != nil {
			panic(err)
		} else if csv {
			fmt.Println(weave.ToCSV(data, columns))
		} else { // default output
			//fmt.Println(weave.ToTable(data, columns))
			// TODO
		}
	}

	// define list flags
	// TODO

	cmd := NewActionCommand(use, short, long, aliases, runFunc)
	cmd.Flags().Bool("csv", false, "output results as a csv")
	return cmd
}
