package list

import (
	"gwcli/action"
	"gwcli/clilog"
	"gwcli/utilities/scaffold/scaffoldlist"

	grav "github.com/gravwell/gravwell/v3/client"
	"github.com/spf13/pflag"

	"github.com/gravwell/gravwell/v3/client/types"
)

var (
	short string = "list installed and staged kits"
	long  string = "lists kits available to your user" +
		"(or all kits on the system, via the --all flag if you are an admin)"
	defaultColumns []string = []string{"UUID", "KitState.Name", "KitState.Description", "KitState.Version"}
)

func NewKitsListAction() action.Pair {
	return scaffoldlist.NewListAction(short, long, defaultColumns,
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
		clilog.LogFlagFailedGet("all", err)
	} else if all {
		return c.AdminListKits()
	}

	return c.ListKits()
}
