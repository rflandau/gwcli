package list

import (
	"gwcli/action"
	"gwcli/clilog"
	"gwcli/connection"
	"gwcli/utilities/scaffold"

	grav "github.com/gravwell/gravwell/v3/client"
	"github.com/spf13/pflag"

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
		types.SearchMacro{}, listMacros, flags)
}

func flags() pflag.FlagSet {
	addtlFlags := pflag.FlagSet{}
	addtlFlags.Bool("all", false, "(admin-only) Fetch all macros on the system."+
		" Supercedes --group. Ignored if you are not an admin.")
	addtlFlags.Int32("group", 0, "Fetches all macros shared with the given grou id.")
	return addtlFlags
}

func listMacros(c *grav.Client, fs *pflag.FlagSet) ([]types.SearchMacro, error) {
	myinfo, err := connection.Client.MyInfo()
	if err != nil {
		return nil, err
	}
	if all, err := fs.GetBool("all"); err != nil {
		clilog.Writer.Errorf("failed to fetch '--all':%v\ndefaulting to false", err)
	} else if all {
		return c.GetAllMacros()
	}
	if gid, err := fs.GetInt32("group"); err != nil {
		clilog.Writer.Errorf("failed to fetch '--group':%v\nignoring", err)
	} else if gid != 0 {
		return c.GetGroupMacros(gid)
	}

	return c.GetUserMacros(myinfo.UID)
}
