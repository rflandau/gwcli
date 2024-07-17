package datascope

/**
 * The Table Tab is able to properly represent tabular data that the results tab would jumble.
 * Meant to represent results returned from GetTableResults (per the table renderer).
 */

import (
	"fmt"
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
		rd["index"] = indexStyle.Render(strconv.Itoa(i + 1))
		for j, c := range cells {
			rd[strconv.Itoa(j+1)] = c
		}
		// add the completed row to the list of rows
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

	// check for column resize keys
	if msg, ok := msg.(tea.KeyMsg); ok { // only care if alt was held
		clilog.Writer.Debugf("Key match: %v", msg)
		switch msg.String() {
		// size increases
		case "alt+1":
			s.table.alterColumnSize(1, true)
		case "alt+2":
			s.table.alterColumnSize(2, true)
		case "alt+3":
			s.table.alterColumnSize(3, true)
		case "alt+4":
			s.table.alterColumnSize(4, true)
		case "alt+5":
			s.table.alterColumnSize(5, true)
		case "alt+6":
			s.table.alterColumnSize(6, true)
		case "alt+7":
			s.table.alterColumnSize(7, true)
		case "alt+8":
			s.table.alterColumnSize(8, true)
		case "alt+9":
			s.table.alterColumnSize(9, true)
		case "alt+0":
			s.table.alterColumnSize(0, true)

		// size decreases
		case "alt+!":
			s.table.alterColumnSize(1, false)
		case "alt+@":
			s.table.alterColumnSize(2, false)
		case "alt+#":
			s.table.alterColumnSize(3, false)
		case "alt+$":
			s.table.alterColumnSize(4, false)
		case "alt+%":
			s.table.alterColumnSize(5, false)
		case "alt+^":
			s.table.alterColumnSize(6, false)
		case "alt+&":
			s.table.alterColumnSize(7, false)
		case "alt+*":
			s.table.alterColumnSize(8, false)
		case "alt+(":
			s.table.alterColumnSize(9, false)
		case "alt+)":
			s.table.alterColumnSize(0, false)
		}
	}

	s.table.vp, cmd = s.table.vp.Update(msg)

	return cmd
}

// alters the flex factor of the columns corresponding to the given number key.
// Treats a 0 as a ten.
func (tt *tableTab) alterColumnSize(numKey uint, increase bool) {
	var colCount uint = uint(len(tt.columns))
	// treat 0 as 10
	if numKey == 0 {
		numKey = 10
	}

	// only increase the size of a column that exists
	if colCount > numKey {
		var newFF int = tt.columns[numKey-1].FlexFactor()
		if increase {
			newFF += 1
		} else {
			newFF -= 1
			if newFF < 0 {
				newFF = 0
			}
		}

		tt.columns[numKey-1] = table.NewFlexColumn(
			tt.columns[numKey-1].Key(),
			tt.columns[numKey-1].Title(),
			newFF)
		tt.tbl = tt.tbl.WithColumns(tt.columns)
		tt.vp.SetContent(tt.tbl.View())
		clilog.Writer.Debugf("installed new columns with an increased size")
	}
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

var tableShortHelp = stylesheet.GreyedOutStyle.Render(
	fmt.Sprintf("%v scroll • home: jump top • end: jump bottom\n"+
		"alt+[1-9]: increase column size • shift+alt+[1-9]: decrease column size\n"+
		"tab: cycle • esc: quit", stylesheet.UpDown),
)

// Draw and return a footer for the viewport
func (tt *tableTab) renderFooter() string {
	var alignerSty = lipgloss.NewStyle().Width(tt.vp.Width).AlignHorizontal(lipgloss.Center)
	return lipgloss.JoinVertical(lipgloss.Center,
		scrollPercentLine(tt.vp.Width, tt.vp.ScrollPercent()),
		alignerSty.Render(tableShortHelp))
}