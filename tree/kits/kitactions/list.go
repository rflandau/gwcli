package kitactions

import (
	"gwcli/action"
	"gwcli/treeutils"

	grav "github.com/gravwell/gravwell/v3/client"

	"github.com/gravwell/gravwell/v3/client/types"
)

var (
	use            string   = "list"
	short          string   = "List all installed and staged kits"
	long           string   = "..."
	aliases        []string = []string{}
	defaultColumns []string = []string{"UID", "NAME", "GLOBAL", "VERSION"}
)

func NewListCmd() action.Pair {
	cmd, la := treeutils.NewListCmd(use, short, long, aliases, defaultColumns, types.IdKitState{}, ListKits)
	return treeutils.GenerateAction(cmd, &la)
}

// Retrieve and return array of kit structs via gravwell client
func ListKits(c *grav.Client) ([]types.IdKitState, error) {
	return c.ListKits()
}
