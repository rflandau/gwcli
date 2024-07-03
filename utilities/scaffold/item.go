package scaffold

import (
	"fmt"
	"gwcli/stylesheet"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// the base functions a delete action must provide on the type it wants deleted
type Item[I id_t] interface {
	ID() I
	FilterValue() string // value to compare against
	String() string      // how the record should be displayed in the list
}

var itemStyle = stylesheet.Composable.Unfocused.PaddingLeft(2)
var selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(stylesheet.PrimaryColor)

// the item delegate defines display format of an item in the list
type itemDelegate[I id_t] struct{}

func (id itemDelegate[I]) Height() int                             { return 2 }
func (id itemDelegate[I]) Spacing() int                            { return 1 }
func (id itemDelegate[I]) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (id itemDelegate[I]) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(Item[I])
	if !ok {
		return
	}

	//str := fmt.Sprintf("%d. $%s --> %s\n  %s", index+1, i.Name, i.Expansion, i.Description)
	str := fmt.Sprintf("%d. %s", index+1, i.String())

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}
