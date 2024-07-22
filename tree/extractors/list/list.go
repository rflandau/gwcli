package list

import (
	"gwcli/action"
	"gwcli/clilog"
	"gwcli/utilities/scaffold/scaffoldlist"

	"github.com/google/uuid"
	grav "github.com/gravwell/gravwell/v3/client"
	"github.com/gravwell/gravwell/v3/client/types"
	"github.com/spf13/pflag"
)

var (
	short          string   = ""
	long           string   = ""
	aliases        []string = []string{}
	defaultColumns []string = []string{"UID", "UUID", "Name", "Desc"}
)

func NewExtractorsListAction() action.Pair {
	return scaffoldlist.NewListAction(short, long, aliases, defaultColumns,
		types.AXDefinition{}, list, flags)
}

func flags() pflag.FlagSet {
	addtlFlags := pflag.FlagSet{}
	addtlFlags.String("uuid", uuid.Nil.String(), "Fetches extraction by uuid.")
	return addtlFlags
}

func list(c *grav.Client, fs *pflag.FlagSet) ([]types.AXDefinition, error) {
	if id, err := fs.GetString("uuid"); err != nil {
		clilog.Writer.Errorf("failed to fetch '--uuid':%v\nignoring", err)
	} else {
		uid, err := uuid.Parse(id)
		if err != nil {
			return nil, err
		}
		if uid != uuid.Nil {
			clilog.Writer.Infof("Fetching ax with uuid %v", uid)
			d, err := c.GetExtraction(id)
			return []types.AXDefinition{d}, err
		}
		// if uid was nil, move on to normal get-all
	}

	return c.GetExtractions()
}