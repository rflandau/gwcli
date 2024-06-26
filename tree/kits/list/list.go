package list

import (
	"gwcli/action"
	"gwcli/utilities/scaffold"

	grav "github.com/gravwell/gravwell/v3/client"
	"github.com/spf13/pflag"

	"github.com/gravwell/gravwell/v3/client/types"
)

var (
	short          string   = "List installed and staged kits"
	long           string   = "..."
	aliases        []string = []string{}
	defaultColumns []string = []string{"UUID", "KitState.Name", "KitState.Description", "KitState.Version"}
)

func NewListCmd() action.Pair {
	addtlFlags := pflag.FlagSet{}
	addtlFlags.Bool("all", false, "(admin-only) Fetch all kits on the system."+
		"Ignored if you are not an admin. ")

	return scaffold.NewListCmd(short, long, aliases, defaultColumns,
		types.IdKitState{}, ListKits, &addtlFlags)
}

// Retrieve and return array of kit structs via gravwell client
func ListKits(c *grav.Client, flags *pflag.FlagSet) ([]types.IdKitState, error) {
	return c.ListKits()
}
