package delete

import (
	"fmt"
	"gwcli/stylesheet"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gravwell/gravwell/v3/client/types"
)

type item types.SearchMacro

// the string value used to compare against a user-given filter to determine eligibility
func (i item) FilterValue() string { return i.Name }

var itemStyle = stylesheet.Composable.Unfocused.PaddingLeft(2)
var selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(stylesheet.PrimaryColor)

type itemDelegate struct{}

func (id itemDelegate) Height() int                             { return 2 }
func (id itemDelegate) Spacing() int                            { return 1 }
func (id itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (id itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. $%s --> %s\n  %s", index+1, i.Name, i.Expansion, i.Description)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}
