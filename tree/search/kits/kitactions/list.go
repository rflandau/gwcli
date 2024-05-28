package kitactions

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

var (
	use     string   = "list"
	short   string   = "List all installed and staged kits"
	long    string   = "..."
	aliases []string = []string{}
)

func NewListCmd() action.Pair {
	return treeutils.GenerateAction(treeutils.NewActionCommand(use, short, long, aliases, run), Kitlist)
}

func run(_ *cobra.Command, _ []string) {
	fmt.Println(listKits())
}

/**
 * Given a kit, returns a slice representing a single row.
 * Format: UID | Global | Name | Version
 */
func rowKit(kit types.IdKitState) []string {
	rowStr := fmt.Sprintf("%v|%v|%v|%v", kit.UID, kit.Name, kit.Global, kit.Version)
	return strings.Split(rowStr, "|")

}

func listKits() string {
	var header []string = []string{"UID", "NAME", "GLOBAL", "VERSION"}

	kits, err := connection.Client.ListKits()
	if err != nil {
		panic(err)
	}
	var kitCount int = len(kits)
	var rows [][]string = make([][]string, kitCount)
	for i := 0; i < kitCount; i++ {
		rows[i] = rowKit(kits[i])
	}

	return stylesheet.Table(header, rows)
}

//#region actor implementation

type kitlist struct {
	done bool
}

var Kitlist action.Model = &kitlist{done: false}

func (k *kitlist) Update(msg tea.Msg) tea.Cmd {
	k.done = true

	return tea.Println(listKits())
}

func (k *kitlist) View() string {
	return ""
}

func (k *kitlist) Done() bool {
	return k.done
}

func (k *kitlist) Reset() error {
	k.done = false
	return nil
}
