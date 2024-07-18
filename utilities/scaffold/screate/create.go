package screate

import (
	"gwcli/action"
	"gwcli/clilog"
	"gwcli/treeutils"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// keys mapped to their fields, how the set is usually passed around and required
type FieldMap = map[string]Field

// signature the supplied creation function must match
type CreateFunc func(FieldMap) (string, error)

func NewCreateAction(aliases []string, singular string,
	fields FieldMap,
	create CreateFunc) action.Pair {
	// pull flags from provided fields
	const mappedString = "mapped field %v (key: %v) to %v flag %v"
	var flags pflag.FlagSet
	for k, f := range fields {
		// extract usable values
		flagName := strings.Replace(f.Title, " ", "-", -1)

		switch f.Type {
		case Text:
			flags.String(flagName, f.Value, f.Usage)
			clilog.Writer.Debugf(mappedString, f.Title, k, "string", flagName)
		default:
			panic("developer error: unknown field type: " + f.Type)
		}
	}

	cmd := treeutils.NewActionCommand(
		"create",
		"create a "+singular,
		"create a new "+singular,
		aliases,
		func(c *cobra.Command, s []string) {

			//
		})

	// attach mined flags to cmd

	return treeutils.GenerateAction(cmd, nil)
}

// base flagset
func flags() pflag.FlagSet {
	return pflag.FlagSet{}
}

//#region Field

type FieldType = string

const (
	Text FieldType = "text"
)

// A field defines a single data point that will be passed to the create function
type Field struct {
	Required        bool // this field must be populated prior to calling createFunc
	Title           string
	StringValidator func(s string) error // validator to run on a text input
	Type            FieldType            // type of field, dictating how it is presented to the user
	Value           string               // user entered value
	Usage           string               // flag usage displayed via -h
}

// Returns a new field with only the required fields. Defaults to a Text type.
func NewField(req bool, Title string) Field {
	return Field{Required: req, Title: Title, Type: Text}
}

//#endregion

//#region interactive mode (model) implementation

type mode uint // state of the interactive application

const (
	inputting mode = iota
	quitting
)

type createModel struct {
	mode mode

	singular string
}

//#endregion

/*
Creates needs to know the fields to present to a user for population.
For each field, we need to know:
1. field name
2. optional or required?
3. corresponding flag for non-interactive input
4. an optional validation function
We need to be able to pass populated fields, once all requireds are filled,
back to the implementation to contort and pass to a create function.
This create function must be able to return an error and, ideally,
a string validation error.
However, we cannot pre-declare the function signature until we know how the
create data will be stored.

Field likely need to be stored in a map (string -> struct).
This will also allow me to bolt additional features onto the struct easily.
*/
