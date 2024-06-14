package systemsactions

import (
	"fmt"
	"gwcli/action"
	"gwcli/group"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewHardwareList() action.Pair {
	cmd := &cobra.Command{
		Use:     "hardware",
		Short:   "Display information about the hardware underlying the instance",
		Long:    "...",
		Aliases: []string{"hw"},
		GroupID: group.ActionID,
		//PreRun: ,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("action called") // TODO
		},
	}
	return action.Pair{cmd, HWlist}
}

//#region actor implementation

type hwlist struct {
	done bool
}

var HWlist action.Model = &hwlist{done: false}

func (k *hwlist) Update(msg tea.Msg) tea.Cmd {
	k.done = true
	return nil
}

func (k *hwlist) View() string {
	return ""
}

func (k *hwlist) Done() bool {
	return true
}

func (k *hwlist) Reset() error {
	k.done = false
	return nil
}

func (k *hwlist) SetArgs(*pflag.FlagSet, []string) (invalid string, onStart []tea.Cmd, err error) {
	return "", nil, nil
}
