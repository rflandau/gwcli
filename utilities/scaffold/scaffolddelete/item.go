package scaffolddelete

import (
	"fmt"
	"gwcli/stylesheet"
	"gwcli/stylesheet/colorizer"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// the base functions a delete action must provide on the type it wants deleted
type Item[I id_t] interface {
	ID() I               // value passed to the delete function
	FilterValue() string // value to compare against for filtration
	String() string      // how the record should be displayed in the list
}

var itemStyle = stylesheet.Composable.Unfocused.PaddingLeft(2)
var selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(stylesheet.PrimaryColor)

const (
	defaultItemHeight  = 2
	defaultItemSpacing = 1
)

// the item delegate defines display format of an item in the list
type defaultDelegate[I id_t] struct {
	height     int
	spacing    int
	renderFunc func(w io.Writer, m list.Model, index int, listItem list.Item)
}

func (d defaultDelegate[I]) Height() int                           { return d.height }
func (d defaultDelegate[I]) Spacing() int                          { return d.spacing }
func (defaultDelegate[I]) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (dd defaultDelegate[I]) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	dd.renderFunc(w, m, index, listItem)
}

// default renderFunc used by the delegate if not overwritten by WithRender()
func defaultRender[I id_t](w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(Item[I])
	if !ok {
		return
	}

	str := fmt.Sprintf("%s%s. %s",
		colorizer.Pip(uint(index), uint(m.Index())),
		colorizer.Index(index+1),
		i.String())
	fmt.Fprint(w, str)
}

// modifiers on the item delegate, usable by implementations of scaffolddelete
type DelegateOption[I id_t] func(*defaultDelegate[I])

// Alter the number of lines allocated to each item.
// Height should be set equal to the lipgloss.Height of your Item.String.
// Values above or below that can have... unpredictable... results.
func WithHeight[I id_t](h int) DelegateOption[I] {
	return func(dd *defaultDelegate[I]) { dd.height = h }
}

// Alter the number of lines between each item
func WithSpacing[I id_t](s int) DelegateOption[I] {
	return func(dd *defaultDelegate[I]) { dd.spacing = s }
}

// Alter how each item is displayed in the list of delete-able items
func WithRender[I id_t](f func(w io.Writer, m list.Model, index int, listItem list.Item)) DelegateOption[I] {
	return func(dd *defaultDelegate[I]) { dd.renderFunc = f }
}
