package kitactions

import (
	"fmt"
	"gwcli/action"
	"gwcli/connection"
	"gwcli/treeutils"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/gravwell/gravwell/v3/client/types"

	"github.com/spf13/cobra"
)

var (
	use     string   = "list"
	short   string   = "List all installed and staged kits"
	long    string   = "..."
	aliases []string = []string{}
)

var (
	headerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("99")).
			AlignHorizontal(lipgloss.Center).AlignVertical(lipgloss.Center)
	baseRowStyle = lipgloss.NewStyle().Padding(0, 1).Width(20)
	EvenRowStyle = baseRowStyle.Foreground(lipgloss.Color("CC22CC"))
	OddRowStyle  = baseRowStyle.Foreground(lipgloss.Color("FF77FF"))
)

var columnsFormat = "%v|%v|%v|%v"

func NewListCmd() action.Pair {
	return treeutils.GenerateAction(use, short, long, aliases, run, Kitlist)
}

func run(_ *cobra.Command, _ []string) {
	fmt.Println(listKits())
}

/**
 * Given a kit, returns a slice representing a single row.
 * Format: UID | Global | Name | Version
 */
func rowKit(kit types.IdKitState) []string {
	rowStr := fmt.Sprintf(columnsFormat, kit.UID, kit.Name, kit.Global, kit.Version)
	return strings.Split(rowStr, "|")

}

func listKits() string {
	// style table
	tbl := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("99"))).
		StyleFunc(func(row, col int) lipgloss.Style {
			switch {
			case row == 0:
				return headerStyle
			case row%2 == 0:
				return EvenRowStyle
			default:
				return OddRowStyle
			}
		}).
		Headers(strings.Split(fmt.Sprintf(columnsFormat, "UID", "NAME", "GLOBAL", "VERSION"), "|")...).
		Border(lipgloss.DoubleBorder()).
		BorderRow(false) //.Width(80)

	kits, err := connection.Client.ListKits()
	if err != nil {
		panic(err)
	}
	for _, k := range kits {
		tbl.Row(rowKit(k)...)
	}

	return tbl.Render()
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
