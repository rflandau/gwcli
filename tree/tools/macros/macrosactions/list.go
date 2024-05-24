package macrosactions

import (
	"fmt"
	"gwcli/action"
	"gwcli/group"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func GenerateAction() action.Pair {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all installed and staged kits",
		Long:    "...",
		Aliases: []string{},
		GroupID: group.ActionID,
		//PreRun: ,
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println(listMacros())
		},
	}

	return action.Pair{cmd, List}

}

func listMacros() string {
	return ""
}

//#region actor implementation

type list struct {
	done bool
}

var List action.Model = &list{done: false}

func (k *list) Update(msg tea.Msg) tea.Cmd {
	k.done = true

	return nil
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
