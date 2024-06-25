package kitactions

import (
	"gwcli/action"
	"gwcli/utilities/scaffold"

	grav "github.com/gravwell/gravwell/v3/client"

	"github.com/gravwell/gravwell/v3/client/types"
)

var (
	short          string   = "List all installed and staged kits"
	long           string   = "..."
	aliases        []string = []string{}
	defaultColumns []string = []string{"UUID", "KitState.Name", "KitState.Description", "KitState.Version"}
)

func NewListCmd() action.Pair {
	return scaffold.NewListCmd(short, long, aliases, defaultColumns, types.IdKitState{}, ListKits)
}

// Retrieve and return array of kit structs via gravwell client
func ListKits(c *grav.Client) ([]types.IdKitState, error) {
	return c.ListKits()
}
