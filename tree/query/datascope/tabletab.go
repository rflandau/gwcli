package datascope

import (
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
		columns[i] = table.Column{Title: c, Width: lipgloss.Width(c)}
	}
	// build rows list
	var rows []table.Row = make([]table.Row, len(data)-1)
	for i, r := range data[1:] {
		rows[i] = strings.Split(r, ",")
	}

	tbl := table.New(table.WithColumns(columns),
		table.WithRows(rows))
	tbl.Focus()

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
	// TODO calculate footer height
	tt.tbl.SetHeight(clippedHeight)
	tt.tbl.SetWidth(rawWidth)
	// re-size columns
	colWidth := rawWidth / len(tt.columns)

	for i := range tt.columns {
		tt.columns[i].Width = colWidth
	}

	tt.tbl.SetColumns(tt.columns)

	tt.ready = true
}

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))
