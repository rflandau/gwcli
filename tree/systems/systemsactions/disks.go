package systemsactions

import (
	"fmt"
	"gwcli/action"
	"gwcli/group"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func NewDiskInfo() action.Pair {
	cmd := &cobra.Command{
		Use:     "disks",
		Short:   "Display information about the disks underlying the instance",
		Long:    "...",
		Aliases: []string{"disk"},
		GroupID: group.ActionID,
		//PreRun: ,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("action called") // TODO
		},
	}
	return action.Pair{cmd, HWlist}
}

type diskInfo struct {
	done bool
}

var DiskInfo action.Model = &diskInfo{done: false}

func (k *diskInfo) Update(msg tea.Msg) tea.Cmd {
	k.done = true
	return nil
}

func (k *diskInfo) View() string {
	return ""
}

func (k *diskInfo) Done() bool {
	return true
}

func (k *diskInfo) Reset() error {
	k.done = false
	return nil
}

func (k *diskInfo) SetArgs([]string) (bool, error) {
	return true, nil
}
