package create

import (
	"gwcli/action"
	"gwcli/connection"
	"gwcli/stylesheet"
	"gwcli/utilities/scaffold/scaffoldcreate"
	"strings"

	"github.com/gravwell/gravwell/v3/client/types"
	"github.com/spf13/pflag"
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
			Usage:         stylesheet.FlagUsageMacroExpansion,
			Type:          scaffoldcreate.Text,
			FlagName:      stylesheet.FlagNameMacroExpansion,
			FlagShorthand: 'e',
			DefaultValue:  "",
			Order:         80,
		},
	}

	return scaffoldcreate.NewCreateAction("macro", fields, create, nil)
}

func create(_ scaffoldcreate.Config, vals scaffoldcreate.Values, _ *pflag.FlagSet) (any, string, error) {
	sm := types.SearchMacro{}
	// all three fields are required, no need to nil-check them
	sm.Name = strings.ToUpper(vals["name"])
	sm.Description = vals["desc"]
	sm.Expansion = vals["exp"]

	id, err := connection.Client.AddMacro(sm)
	return id, "", err

}
