package kitactions

import (
	"fmt"
	"gwcli/action"
	"gwcli/connection"
	"gwcli/stylesheet"
	"gwcli/treeutils"
	"strings"

	grav "github.com/gravwell/gravwell/v3/client"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gravwell/gravwell/v3/client/types"
)

var (
	use     string   = "list"
	short   string   = "List all installed and staged kits"
	long    string   = "..."
	aliases []string = []string{}
)

func NewListCmd() action.Pair {
	cmd := treeutils.NewListCmd(use, short, long, aliases, ListKits)
	return treeutils.GenerateAction(cmd, Kitlist)
}

// Retrieve and return array of kit structs via gravwell client
func ListKits(c *grav.Client) ([]types.IdKitState, error) {
	return c.ListKits()
}

/**
 * Given a kit, returns a slice representing a single row.
 * Format: UID | Global | Name | Version
 */
func rowKit(kit types.IdKitState) []string {
	rowStr := fmt.Sprintf("%v|%v|%v|%v", kit.UID, kit.Name, kit.Global, kit.Version)
	return strings.Split(rowStr, "|")

}

// TODO convert this to a weave.ToTable
func listKits(kits []types.IdKitState) string {
	var header []string = []string{"UID", "NAME", "GLOBAL", "VERSION"}

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

	data, err := connection.Client.ListKits()
	if err != nil {
		panic(err)
	}
	return tea.Println(listKits(data))
}

func (k *kitlist) View() string {
	// no action required; line is output as history in Update
	return ""
}

func (k *kitlist) Done() bool {
	return k.done
}

func (k *kitlist) Reset() error {
	k.done = false
	return nil
}
