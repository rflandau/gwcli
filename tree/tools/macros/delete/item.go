package delete

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gravwell/gravwell/v3/client/types"
)

type item types.SearchMacro

func (i item) Title() string       { return i.Name }
func (i item) Desc() string        { return i.Description }
func (i item) Exp() string         { return i.Expansion }
func (i item) FilterValue() string { return i.Name }

var itemStyle = lipgloss.NewStyle().PaddingLeft(4)
var selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))

type itemDelegate struct{}

func (id itemDelegate) Height() int                             { return 2 }
func (id itemDelegate) Spacing() int                            { return 1 }
func (id itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (id itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i.Name)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}
