package datascope

import (
	"gwcli/clilog"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
)

type tableTab struct {
	columns []table.Column // once installed, tbl.columns is not externally accessible
	tbl     table.Model
	ready   bool
}

func initTableTab(data []string) tableTab {
	// build columns list
	strcols := strings.Split(data[0], ",")
	colCount := len(strcols)
	var columns []table.Column = make([]table.Column, colCount)
	for i, c := range strcols {
		// columns display the given column name, but are mapped by number
		columns[i] = table.NewFlexColumn(strconv.Itoa(i), c, 1)
		clilog.Writer.Debugf("Added column %v (key: %v)", columns[i].Title(), columns[i].Key())
	}
	// build rows list
	var rows []table.Row = make([]table.Row, len(data)-1)
	for i, r := range data[1:] {
		cells := strings.Split(r, ",")
		// map each row cell to its column
		rd := table.RowData{}
		for j, c := range cells {
			rd[strconv.Itoa(j)] = c
		}
		// add the completed row to the list of rows
		clilog.Writer.Debugf("Adding row %v", rd)
		rows[i] = table.NewRow(rd)
	}

	tbl := table.New(columns).
		WithRows(rows).
		Focused(true).
		WithMultiline(true).WithStaticFooter("footer")

	return tableTab{
		columns: columns,
		tbl:     tbl,
	}
}

func updateTable(s *DataScope, msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	s.table.tbl, cmd = s.table.tbl.Update(msg)
	return cmd
}

func viewTable(s *DataScope) string {
	if !s.table.ready {
		return "\nInitializing..."
	}
	return baseStyle.Render(s.table.tbl.View())
}

// recalculate and update the size parameters of the table.
// The clipped height is the height available to the table tab (height - tabs height).
func (tt *tableTab) recalculateSize(rawWidth, clippedHeight int) {
	tt.tbl = tt.tbl.WithMaxTotalWidth(rawWidth).
		WithTargetWidth(rawWidth).
		WithPageSize(clippedHeight - 8) // 8 is extra padding due to the margins of the table itself
	tt.ready = true
}

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))
