package datascope

/**
 * The Table Tab is able to properly represent tabular data that the results tab would jumble.
 * Meant to represent results returned from GetTableResults (per the table renderer).
 */

import (
	"gwcli/clilog"
	"gwcli/stylesheet"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
)

const flexFactor = 5 // target ratio: other column width : index column width (1)

var sep = ","

type tableTab struct {
	vp      viewport.Model
	columns []table.Column // once installed, tbl.columns is not externally accessible
	rows    []string       // save off data minus header for easy access by dl tab
	tbl     table.Model
	ready   bool
}

// Initializes the table tab, setting up the viewport and tabulating the data.
//
// ! Assumes data[0] is the columns headers
func initTableTab(data []string) tableTab {
	vp := NewViewport() // spawn the vp wrapper of the table

	// build columns list, with the index column prefixed
	strcols := strings.Split(data[0], sep)
	colCount := len(strcols) + 1
	var columns []table.Column = make([]table.Column, colCount)
	// set index column
	columns[0] = table.NewFlexColumn("index", "#", 1)
	for i, c := range strcols {
		// columns display the given column name, but are mapped by number
		columns[i+1] = table.NewFlexColumn(strconv.Itoa(i+1), c, flexFactor)
		clilog.Writer.Debugf("Added column %v (key: %v)", columns[i].Title(), columns[i].Key())
	}
	// build rows list
	var rows []table.Row = make([]table.Row, len(data)-1)
	for i, r := range data[1:] {
		cells := strings.Split(r, sep)
		// map each row cell to its column
		rd := table.RowData{}
		// prepend the index column
		rd["index"] = i + 1
		for j, c := range cells {
			rd[strconv.Itoa(j+1)] = c
		}
		// add the completed row to the list of rows
		clilog.Writer.Debugf("Adding row %v", rd)
		rows[i] = table.NewRow(rd)
	}

	tbl := table.New(columns).
		WithRows(rows).
		Focused(true).
		WithMultiline(true).
		WithStaticFooter("END OF DATA").
		WithRowStyleFunc(func(rsfi table.RowStyleFuncInput) lipgloss.Style {
			if rsfi.Index%2 == 0 {
				return evenEntryStyle
			}
			return oddEntryStyle
		}).
		HeaderStyle(stylesheet.Tbl.HeaderCells)
		// NOTE: As of evertras-table v0.16.1,
		// the borders cannot be styled (only their runes changed.)

	// display the table within the viewport
	vp.SetContent(tbl.View())

	return tableTab{
		vp:      vp,
		rows:    data[1:],
		columns: columns,
		tbl:     tbl,
	}
}

// Pass messages to the viewport. The underlying table does not get updated.
func updateTable(s *DataScope, msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	// check for keybinds not directly supported by the viewport
	if viewportAddtlKeys(msg, &s.table.vp) {
		return nil
	}

	s.table.vp, cmd = s.table.vp.Update(msg)

	return cmd
}

func viewTable(s *DataScope) string {
	if !s.table.ready {
		return "\nInitializing..."
	}
	return s.table.vp.View() + "\n" + s.table.renderFooter()
}

// recalculate and update the size parameters of the table.
// The clipped height is the height available to the table tab (height - tabs height).
func (tt *tableTab) recalculateSize(rawWidth, clippedHeight int) {
	tt.tbl = tt.tbl.WithMaxTotalWidth(rawWidth).
		WithTargetWidth(rawWidth) //.WithPageSize(clippedHeight - 13) // 8 is extra padding due to the margins of the table itself
	tt.vp.Width = rawWidth
	tt.vp.Height = clippedHeight - lipgloss.Height(tt.renderFooter())
	tt.vp.SetContent(tt.tbl.View())
	tt.ready = true
}

// Draw and return a footer for the viewport
func (tt *tableTab) renderFooter() string {
	return scrollPercentLine(tt.vp.Width, tt.vp.ScrollPercent())
}
