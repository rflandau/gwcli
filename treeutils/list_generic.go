/**
 * Helper functions and generic struct.
 * Intended to be boilder plate for specific list implementations.
 */

package treeutils

import (
	"errors"
	"fmt"
	"gwcli/clilog"
	"gwcli/connection"
	"gwcli/weave"
	"reflect"

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

//#region errors

const (
	ErrNotAStruct string = "given value is not a struct or pointer to a struct"
	ErrIsNil      string = "given value is nil"
)

//#endregion

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

// Returns a list of all fields in the struct *definition*, as they are ordered
// internally
func StructFields(st any) (columns []string, err error) {
	if st == nil {
		return nil, errors.New(ErrIsNil)
	}
	to := reflect.TypeOf(st)
	if to.Kind() == reflect.Pointer { // dereference
		to = to.Elem()
	}
	if to.Kind() != reflect.Struct { // prerequisite
		return nil, errors.New(ErrNotAStruct)
	}
	numFields := to.NumField()
	columns = []string{}

	// for each field
	//	if the field is not a struct, append it to the columns
	//	if the field is a struct, repeat

	for i := 0; i < numFields; i++ {
		columns = append(columns, innerStructFields("", to.Field(i))...)
	}

	return columns, nil
}

// innerStructFields is a helper function for StructFields, returning the
// qualified name of the given field or the list of qualified names of its
// children, if a struct.
// Operates recursively on the given field if it is a struct.
// Operates down the struct, in field-order.
func innerStructFields(qualification string, field reflect.StructField) []string {
	var columns []string = []string{}
	if field.Type.Kind() == reflect.Struct {
		for k := 0; k < field.Type.NumField(); k++ {
			var innerQual string
			if qualification == "" {
				innerQual = field.Name
			} else {
				innerQual = qualification + "." + field.Name
			}

			columns = append(columns, innerStructFields(innerQual, field.Type.Field(k))...)
		}
	} else {
		if qualification == "" {
			columns = append(columns, field.Name)
		} else {
			columns = append(columns, qualification+"."+field.Name)
		}
	}

	return columns
}

// Returns a list of all fields in the struct *definition*
func StructFieldsO(st any) (columns []string) {
	types := reflect.ValueOf(st).Type()
	numFields := types.NumField()
	columns = make([]string, numFields)

	// TODO use FieldByIndex to dig into embedded types and the direct field names

	for i := 0; i < numFields; i++ {
		field := types.Field(i)
		fbi := types.FieldByIndex(field.Index)
		fmt.Printf("{%d}\nfield: %#v\nfieldIndex: %+v\nfbi: %+v", i, field, field.Index, fbi)
		columns[i] = fbi.Name

		//columns[i] = types.Field(i).Name
	}

	return
}
