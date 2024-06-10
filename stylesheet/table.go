package stylesheet

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

const ( // colors
	borders = "#bb9af7"
	header = "#7aa2f7"
	row1 = "#c0caf5"
	row2 = "#9aa5ce"
)

var (
	baseCell = lipgloss.NewStyle().Padding(0, 1).Width(30)

	tbl = struct {
		// cells
		headerCells       lipgloss.Style
		evenCells lipgloss.Style
		oddCells  lipgloss.Style

		// borders
		borderType   lipgloss.Border
		border  lipgloss.Style
		
	}{
		//cells
		headerCells: lipgloss.NewStyle().
			Foreground(lipgloss.Color(header)).
			AlignHorizontal(lipgloss.Center).
			AlignVertical(lipgloss.Center).Bold(true),
		evenCells: baseCell.
			Foreground(lipgloss.Color(row1)),//.Background(lipgloss.Color("#000")),
		oddCells: baseCell.
			Foreground(lipgloss.Color(row2)),
		
		// borders
		borderType:  lipgloss.NormalBorder(),
		border: lipgloss.NewStyle().Foreground(lipgloss.Color(borders)),
		}
)

// Generate a styled table skeleton
func Table() *table.Table {
	tbl := table.New().
		Border(tbl.borderType).
		BorderStyle(tbl.border).
		StyleFunc(func(row, col int) lipgloss.Style {
			switch {
			case row == 0:
				return tbl.headerCells
			case row%2 == 0:
				return tbl.evenCells
			default:
				return tbl.oddCells
			}
		}).BorderRow(true)

	return tbl
}
