package create

import (
	"gwcli/action"
	"gwcli/connection"
	"gwcli/utilities/scaffold/scaffoldcreate"
	"strings"

	"github.com/gravwell/gravwell/v3/client/types"
)

func NewMacroCreateAction() action.Pair {
	n := scaffoldcreate.NewField(true, "name", 100)
	n.FlagShorthand = 'n'
	d := scaffoldcreate.NewField(true, "description", 90)
	d.FlagShorthand = 'd'

	fields := scaffoldcreate.Config{
		"name": n,
		"desc": d,
		"exp": scaffoldcreate.Field{
			Required:      true,
			Title:         "expansion",
			Usage:         "value for the macro to expand to",
			Type:          scaffoldcreate.Text,
			FlagName:      "expansion",
			FlagShorthand: 'e',
			DefaultValue:  "",
			TI: struct {
				Order       int
				Placeholder string
				Validator   func(s string) error
			}{
				Order: 80,
			},
		},
	}

	return scaffoldcreate.NewCreateAction("macro", fields, create)
}

func create(_ scaffoldcreate.Config, vals scaffoldcreate.Values) (any, string, error) {
	sm := types.SearchMacro{}
	// all three fields are required, no need to nil-check them
	sm.Name = strings.ToUpper(vals["name"])
	sm.Description = vals["desc"]
	sm.Expansion = vals["exp"]

	id, err := connection.Client.AddMacro(sm)
	return id, "", err

}
