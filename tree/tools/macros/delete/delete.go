package delete

import (
	"fmt"
	"gwcli/action"
	"gwcli/connection"
	"gwcli/stylesheet"
	"gwcli/utilities/scaffold/scaffolddelete"
	"slices"
	"strings"

	"github.com/gravwell/gravwell/v3/client/types"
)

func NewMacroDeleteAction() action.Pair {
	return scaffolddelete.NewDeleteAction("macro", "macros", del,
		func() ([]scaffolddelete.Item[uint64], error) {
			ms, err := connection.Client.GetUserGroupsMacros()
			if err != nil {
				return nil, err
			}
			slices.SortFunc(ms, func(m1, m2 types.SearchMacro) int {
				return strings.Compare(m1.Name, m2.Name)
			})
			var items = make([]scaffolddelete.Item[uint64], len(ms))
			for i := range ms {
				items[i] = macroItem{
					id:          ms[i].ID,
					name:        ms[i].Name,
					description: ms[i].Description,
					expansion:   ms[i].Expansion,
				}
			}
			return items, nil
		})
}

func del(dryrun bool, id uint64) error {
	if dryrun {
		_, err := connection.Client.GetMacro(id)
		return err
	}
	return connection.Client.DeleteMacro(id)
}

type macroItem struct {
	id          uint64
	name        string
	description string
	expansion   string
}

var _ scaffolddelete.Item[uint64] = macroItem{}

func (mi macroItem) ID() uint64          { return mi.id }
func (mi macroItem) FilterValue() string { return mi.name }
func (mi macroItem) Title() string       { return mi.name }
func (mi macroItem) Details() string {
	return fmt.Sprintf("Expansion: '%v'\n%v",
		stylesheet.Header2Style.Render(mi.expansion), mi.description)
}
