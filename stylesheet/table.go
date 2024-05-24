package stylesheet

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

var (
	baseRowStyle = lipgloss.NewStyle().Padding(0, 1).Width(20)

	tblStyle = struct {
		borderType   lipgloss.Border
		borderStyle  lipgloss.Style
		header       lipgloss.Style
		evenRowStyle lipgloss.Style
		oddRowStyle  lipgloss.Style
	}{
		borderType:  lipgloss.NormalBorder(),
		borderStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("99")),
		header: lipgloss.NewStyle().
			Foreground(lipgloss.Color("99")).
			AlignHorizontal(lipgloss.Center).
			AlignVertical(lipgloss.Center),
		evenRowStyle: baseRowStyle.
			Foreground(lipgloss.Color("CC22CC")),
		oddRowStyle: baseRowStyle.
			Foreground(lipgloss.Color("FF77FF"))}

	baseTbl *table.Table = table.New().
		Border(tblStyle.borderType).
		BorderStyle(tblStyle.borderStyle).
		StyleFunc(func(row, col int) lipgloss.Style {
			switch {
			case row == 0:
				return tblStyle.header
			case row%2 == 0:
				return tblStyle.evenRowStyle
			default:
				return tblStyle.oddRowStyle
			}
		}).BorderRow(false)
)

func Table(header []string, rows [][]string) string {
	tbl := baseTbl // duplicate table style
	// populate data
	tbl.Headers(header...)
	tbl.Rows(rows...)
	return tbl.Render()
}
