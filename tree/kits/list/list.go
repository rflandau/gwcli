package list

import (
	"gwcli/action"
	"gwcli/clilog"
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
	return scaffold.NewListCmd(short, long, aliases, defaultColumns,
		types.IdKitState{}, ListKits, flags)
}

func flags() pflag.FlagSet {
	addtlFlags := pflag.FlagSet{}
	addtlFlags.Bool("all", false, "(admin-only) Fetch all kits on the system."+
		"Ignored if you are not an admin.")

	return addtlFlags
}

// Retrieve and return array of kit structs via gravwell client
func ListKits(c *grav.Client, flags *pflag.FlagSet) ([]types.IdKitState, error) {
	// if --all, use the admin version
	if all, err := flags.GetBool("all"); err != nil {
		clilog.Writer.Errorf("failed to fetch '--all':%v\ndefaulting to false", err)
	} else if all {
		return c.AdminListKits()
	}

	return c.ListKits()
}
