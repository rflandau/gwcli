package delete

import (
	"gwcli/action"
	"gwcli/connection"
	"gwcli/utilities/scaffold"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/gravwell/gravwell/v3/client/types"
)

var (
	use     string   = "delete"
	short   string   = "Delete a macro"
	long    string   = "Delete a macro by id or selection"
	aliases []string = []string{}
)

func NewQueriesScheduledDeleteAction() action.Pair {
	return scaffold.NewDeleteAction(short, long, aliases,
		"query", "queries", del, fetch2)
}

func del(dryrun bool, id int32) error {
	if dryrun {
		_, err := connection.Client.GetScheduledSearch(id)
		return err
	}

	return connection.Client.DeleteScheduledSearch(id)

}

// select and sort the scheduled searches to cursor through
func fetch2() ([]scheduledSearchItem, error) {
	//var items []list.Item
	ss, err := connection.Client.GetScheduledSearchList()
	if err != nil {
		return nil, err
	}
	// sort the results on name
	slices.SortFunc(ss, func(m1, m2 types.ScheduledSearch) int {
		return strings.Compare(m1.Name, m2.Name)
	})
	var items = make([]scheduledSearchItem, len(ss))
	for i := range ss {
		items[i] = scheduledSearchItem{id: ss[i].ID, name: ss[i].Name}
	}

	return items, nil
}

// select and sort the scheduled searches to cursor through
func fetch() ([]list.Item, error) {
	var items []list.Item
	ss, err := connection.Client.GetScheduledSearchList()
	if err != nil {
		return nil, err
	}
	// sort the results on name
	slices.SortFunc(ss, func(m1, m2 types.ScheduledSearch) int {
		return strings.Compare(m1.Name, m2.Name)
	})
	items = make([]list.Item, len(ss))
	for i := range ss {
		items[i] = scheduledSearchItem{id: ss[i].ID, name: ss[i].Name}
	}

	return items, nil
}

type scheduledSearchItem struct {
	id   int32 // the id used to delete an ss
	name string
}

var _ scaffold.Item[int32] = scheduledSearchItem{}

func (ssi scheduledSearchItem) ID() int32 {
	return ssi.id
}

func (ssi scheduledSearchItem) FilterValue() string {
	return ssi.name
}

func (ssi scheduledSearchItem) String() string {
	return ssi.name
}
