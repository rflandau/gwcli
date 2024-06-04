package list

import (
	"fmt"
	"gwcli/action"
	"gwcli/connection"
	"gwcli/stylesheet"
	"gwcli/treeutils"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gravwell/gravwell/v3/client/types"
	"github.com/spf13/cobra"
)

func GenerateListAction() action.Pair {
	return treeutils.GenerateAction(
		treeutils.NewActionCommand("list",
			"list your macros", "list prints out all macros associated to your user.\n"+
				"(NYI) Use the x flag to get all macros system-wide or the y <user>"+
				"parameter to all macros associated to a <user> (if you are an admin)",
			[]string{},
			run),
		List)
}

/* cobra run command for non-interactive usage */
func run(_ *cobra.Command, _ []string) {
	fmt.Println(listMacros())
}

func rowMacros(macro types.SearchMacro) []string {
	rowStr := fmt.Sprintf("%v|%v|%v|%v|%v", macro.ID, macro.Name, macro.Description, macro.Expansion, macro.Labels)
	return strings.Split(rowStr, "|")
}

func listMacros() (string, error) {
	myinfo, err := connection.Client.MyInfo()
	if err != nil {
		return "", err
	}
	macros, err := connection.Client.GetUserMacros(myinfo.UID)
	if err != nil {
		return "", err
	}

	// convert macros to rows
	var macrosCount int = len(macros)
	var rows [][]string = make([][]string, macrosCount)
	for i := 0; i < macrosCount; i++ {
		rows[i] = rowMacros(macros[i])
	}

	return stylesheet.Table([]string{"ID", "NAME", "DESCRIPTION", "EXPANSION", "LABELS"}, rows), nil
}

//#region actor implementation

type list struct {
	done bool
}

var List action.Model = &list{done: false}

func (k *list) Update(msg tea.Msg) tea.Cmd {
	k.done = true

	return tea.Println(listMacros())
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
