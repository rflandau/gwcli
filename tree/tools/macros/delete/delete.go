package delete

import (
	"gwcli/action"
	"gwcli/connection"
	"gwcli/utilities/scaffold"
	"slices"
	"strings"

	"github.com/gravwell/gravwell/v3/client/types"
)

var (
	aliases []string = []string{}
)

func NewMacroDeleteAction() action.Pair {
	return scaffold.NewDeleteAction(aliases, "macro", "macros", del,
		func() ([]scaffold.Item[uint64], error) {
			ms, err := connection.Client.GetUserGroupsMacros()
			if err != nil {
				return nil, err
			}
			slices.SortFunc(ms, func(m1, m2 types.SearchMacro) int {
				return strings.Compare(m1.Name, m2.Name)
			})
			var items = make([]scaffold.Item[uint64], len(ms))
			for i := range ms {
				items[i] = macroItem{id: ms[i].ID, name: ms[i].Name}
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
	id   uint64
	name string
}

var _ scaffold.Item[uint64] = macroItem{}

func (mi macroItem) ID() uint64          { return mi.id }
func (mi macroItem) FilterValue() string { return mi.name }
func (mi macroItem) String() string      { return mi.name }
