package macrosactions

import (
	"fmt"
	"gwcli/action"
	"gwcli/treeutils"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var (
	use     string   = "list"
	short   string   = "List all macros"
	long    string   = "..."
	aliases []string = []string{}
)

func GenerateAction() action.Pair {
	return treeutils.GenerateAction(use, short, long, aliases, run, List)
}

func run(_ *cobra.Command, _ []string) {
	fmt.Println(listMacros())
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
