package datascope

import (
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tableTab struct {
	tbl table.Model
}

func initTableTab() tableTab {
	return tableTab{}
}

func updateTable(s *DataScope, msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	s.table.tbl, cmd = s.table.tbl.Update(msg)
	return cmd
}

func viewTable(s *DataScope) string {
	return baseStyle.Render(s.table.tbl.View()) + "\n"
}

// recalculate and update the size parameters of the table.
// The clipped height is the height available to the table tab (height - tabs height).
func (tt *tableTab) recalculateSize(rawWidth, clippedHeight int) {
	// TODO calculate footer height
	tt.tbl.SetHeight(clippedHeight)
	tt.tbl.SetWidth(rawWidth)
}

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))
