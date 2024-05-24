package create

import (
	"fmt"
	"gwcli/action"
	"gwcli/treeutils"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func GenerateAction() action.Pair {
	return treeutils.GenerateAction("create", "create a new macro", "", []string{}, run, Create)
}

func run(_ *cobra.Command, _ []string) {
	fmt.Println("create macro")
}

func createMacro() {

}

//#region actor implementation

type create struct {
	name string
	done bool
}

var Create action.Model = &create{done: false}

func (k *create) Update(msg tea.Msg) tea.Cmd {
	// spin up a dialogue to take user input

	// TODO

	k.done = true
	return nil
}

func (k *create) View() string {
	return ""
}

func (k *create) Done() bool {
	return k.done
}

func (k *create) Reset() error {
	k.done = false
	return nil
}

//#endregion
