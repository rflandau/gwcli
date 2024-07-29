package delete

import (
	"fmt"
	"gwcli/action"
	"gwcli/connection"
	"gwcli/utilities/scaffold/scaffolddelete"
	"time"
)

func NewDashboardDeleteAction() action.Pair {
	return scaffolddelete.NewDeleteAction("dashboard", "dashboards",
		del, fch)
}

func del(dryrun bool, id uint64) error {
	if dryrun {
		_, err := connection.Client.GetDashboard(id)
		return err
	}
	return connection.Client.DeleteDashboard(id)
}

func fch() ([]scaffolddelete.Item[uint64], error) {
	ud, err := connection.Client.GetUserDashboards(connection.MyInfo.UID)
	if err != nil {
		return nil, err
	}
	// not too important to sort this one
	var items = make([]scaffolddelete.Item[uint64], len(ud))
	for i := range items {
		items[i] = dashItem{
			id:    ud[i].ID,
			title: ud[i].Name,
			details: fmt.Sprintf("Updated: %v\n%s",
				ud[i].Updated.Format(time.RFC822), ud[i].Description),
		}
	}

	return items, nil
}

type dashItem struct {
	id      uint64
	title   string
	details string
}

var _ scaffolddelete.Item[uint64] = dashItem{}

func (di dashItem) ID() uint64          { return di.id }
func (di dashItem) FilterValue() string { return di.title }
func (di dashItem) Title() string       { return di.title }
func (di dashItem) Details() string     { return di.details }
