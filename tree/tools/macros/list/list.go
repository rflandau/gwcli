package list

import (
	"gwcli/action"
	"gwcli/connection"
	"gwcli/utilities/scaffold"

	grav "github.com/gravwell/gravwell/v3/client"

	"github.com/gravwell/gravwell/v3/client/types"
)

var (
	short string = "List your macros"
	long  string = "Prints out all macros associated to your user.\n" +
		"(NYI) Use the x flag to get all macros system-wide or the y <user>" +
		"parameter to all macros associated to a <user> (if you are an admin)"
	aliases        []string = []string{}
	defaultColumns []string = []string{"UID", "Name", "Description", "Expansion"}
)

func NewListCmd() action.Pair {
	return scaffold.NewListCmd(short, long, aliases, defaultColumns,
		types.SearchMacro{}, listMacros)
}

func listMacros(c *grav.Client) ([]types.SearchMacro, error) {
	myinfo, err := connection.Client.MyInfo()
	if err != nil {
		return nil, err
	}
	return c.GetUserMacros(myinfo.UID)
}
