package list

import (
	"gwcli/action"
	"gwcli/connection"
	"gwcli/treeutils"

	tea "github.com/charmbracelet/bubbletea"
	grav "github.com/gravwell/gravwell/v3/client"

	"github.com/gravwell/gravwell/v3/client/types"
)

func NewListCmd() action.Pair {
	cmd := treeutils.NewListCmd("list",
		"list your macros", "list prints out all macros associated to your user.\n"+
			"(NYI) Use the x flag to get all macros system-wide or the y <user>"+
			"parameter to all macros associated to a <user> (if you are an admin)", []string{}, types.SearchMacro{}, listMacros)
	return treeutils.GenerateAction(cmd, List)
}

func listMacros(c *grav.Client) ([]types.SearchMacro, error) {
	myinfo, err := connection.Client.MyInfo()
	if err != nil {
		return nil, err
	}
	return c.GetUserMacros(myinfo.UID)
}

//#region actor implementation

type list struct {
	done bool
}

var List action.Model = &list{done: false}

func (k *list) Update(msg tea.Msg) tea.Cmd {
	k.done = true

	return tea.Println(listMacros(connection.Client))
}

func (k *list) View() string {
	return ""
}

func (k *list) Done() bool {
	return k.done
}

func (k *list) Reset() error {
	k.done = false
	return nil
}

func (k *list) SetArgs([]string) (bool, error) {
	return true, nil
}

//#endregion
