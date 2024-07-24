package list

import (
	"errors"
	"fmt"
	"gwcli/action"
	"gwcli/clilog"
	"gwcli/utilities/scaffold/scaffoldlist"
	"strconv"

	"github.com/google/uuid"
	grav "github.com/gravwell/gravwell/v3/client"
	"github.com/gravwell/gravwell/v3/client/types"
	"github.com/spf13/pflag"
)

var (
	short          string   = "list scheduled queries"
	long           string   = "prints out all scheduled queries."
	aliases        []string = []string{}
	defaultColumns []string = []string{"ID", "Name", "Description", "Duration", "Schedule"}
)

func NewScheduledQueriesListAction() action.Pair {
	return scaffoldlist.NewListAction(short, long, aliases, defaultColumns,
		types.ScheduledSearch{}, listScheduledSearch, flags)
}

func flags() pflag.FlagSet {
	addtlFlags := pflag.FlagSet{}
	addtlFlags.Bool("all", false, "(admin-only) Fetch all scheduled searches on the system."+
		" Supercedes --id. Returns your searches if you are not an admin.")
	addtlFlags.String("id", "", "Fetches the scheduled search associated to the given id."+
		"This id can be a standard, numeric ID or a uuid.")

	return addtlFlags
}

func listScheduledSearch(c *grav.Client, fs *pflag.FlagSet) ([]types.ScheduledSearch, error) {
	if all, err := fs.GetBool("all"); err != nil {
		clilog.Writer.Errorf("failed to fetch '--all':%v\ndefaulting to false", err)
	} else if all {
		return c.GetAllScheduledSearches()
	}
	if untypedID, err := fs.GetString("id"); err != nil {
		clilog.Writer.Errorf("failed to fetch '--id':%v\nignoring", err)
	} else if untypedID != "" {
		// attempt to parse as UUID first
		if uuid, err := uuid.Parse(untypedID); err == nil {
			ss, err := c.GetScheduledSearch(uuid)
			return []types.ScheduledSearch{ss}, err
		}
		// now try as int32
		if i32id, err := strconv.Atoi(untypedID); err == nil {
			ss, err := c.GetScheduledSearch(i32id)
			return []types.ScheduledSearch{ss}, err
		}

		// both have failed, error out
		errString := fmt.Sprintf("failed to parse %v as a uuid or int32 id", untypedID)
		clilog.Writer.Infof(errString)

		return nil, errors.New(errString)
	}
	return c.GetScheduledSearchList()
}
