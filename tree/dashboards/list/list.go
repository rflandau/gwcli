package list

import (
	"fmt"
	"gwcli/action"
	"gwcli/clilog"
	"gwcli/connection"
	"gwcli/stylesheet"
	"gwcli/utilities/scaffold/scaffoldlist"

	grav "github.com/gravwell/gravwell/v3/client"
	"github.com/gravwell/gravwell/v3/client/types"
	"github.com/spf13/pflag"
)

var (
	short          string   = "list dashboards"
	long           string   = "list dashboards available to you and the system"
	aliases        []string = []string{}
	defaultColumns []string = []string{"ID", "Name", "Description"}
)

func NewDashboardsListAction() action.Pair {
	return scaffoldlist.NewListAction(short, long, aliases, defaultColumns,
		types.Dashboard{}, list, flags)
}

func flags() pflag.FlagSet {
	addtlFlags := pflag.FlagSet{}
	addtlFlags.Bool("all", false, fmt.Sprintf(stylesheet.FlagListAllDescFormat, "dashboards"))

	return addtlFlags
}

func list(c *grav.Client, fs *pflag.FlagSet) ([]types.Dashboard, error) {
	if all, err := fs.GetBool("all"); err != nil {
		clilog.LogFlagFailedGet("all", err)
	} else if all {
		return c.GetAllDashboards()
	}
	return c.GetUserDashboards(connection.MyInfo.UID)
}
