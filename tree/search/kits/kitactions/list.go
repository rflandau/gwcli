package kitactions

import (
	"fmt"
	"gwcli/connection"
	"gwcli/treeutils"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/gravwell/gravwell/v3/client/types"

	"github.com/spf13/cobra"
)

var (
	headerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("99")).
			AlignHorizontal(lipgloss.Center).AlignVertical(lipgloss.Center)
	baseRowStyle = lipgloss.NewStyle().Padding(0, 1).Width(20)
	EvenRowStyle = baseRowStyle.Foreground(lipgloss.Color("CC22CC"))
	OddRowStyle  = baseRowStyle.Foreground(lipgloss.Color("FF77FF"))
)

var columnsFormat = "%v|%v|%v|%v"

func NewListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all installed and staged kits",
		Long:    "...",
		Aliases: []string{},
		GroupID: treeutils.ActionID,
		//PreRun: ,
		Run: func(cmd *cobra.Command, args []string) {
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

			fmt.Println(tbl)
		},
	}
	return cmd
}

/**
 * Given a kit, returns a slice representing a single row.
 * Format: UID | Global | Name | Version
 */
func rowKit(kit types.IdKitState) []string {
	rowStr := fmt.Sprintf(columnsFormat, kit.UID, kit.Name, kit.Global, kit.Version)
	return strings.Split(rowStr, "|")

}
