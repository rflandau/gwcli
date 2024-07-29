/*
This package supplies the item delegate used for list bubble implementations (primarily in the
scaffolds) within gwcli. By sharing a single definition, we can ensure lists look and function
identically no matter what action or scaffold is invoking it.
*/
package listsupport

import (
	"fmt"
	"gwcli/stylesheet"
	"gwcli/stylesheet/colorizer"
	"io"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func NewList(items []list.Item, width, height int) list.Model {
	l := list.New(items, list.NewDefaultDelegate(), width, height)
	l.KeyMap = keyMap()
	return l
}

// An entry is an actual item in the list. When an item is retrieved from a list.Model, it should b
// cast to an Entry.
type Entry struct {
	Name    string // the unique name of this item
	Details string // the second (and potentially more) line of this item
}

func (e Entry) FilterValue() string { return e.Name + "\n" + e.Details }

//#region Delegate

// The delegate defines how items are displayed when viewing the list.
type Delegate struct {
	width        int // width of an entry before wrapping
	entryHeight  int // height of the entry. Should be equal to lipgloss.Height
	entrySpacing int // extra padding between entries (+ height)
	// override-able render function.
	// defaults to defaultRender if nil
	renderFunc func(w io.Writer, m list.Model, index int, listItem list.Item)
}

func newDelegate(dopts ...DelegateOption) Delegate {
	d := Delegate{
		width:        80,
		entryHeight:  2,
		entrySpacing: 1,
		renderFunc:   defaultRender,
	}

	// apply options
	for _, dopt := range dopts {
		if dopt != nil {
			dopt(&d)
		}
	}

	return d
}

func (d Delegate) Height() int                           { return d.entryHeight }
func (d Delegate) Spacing() int                          { return d.entrySpacing }
func (Delegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d Delegate) Render(w io.Writer, m list.Model, index int, itm list.Item) {
	d.renderFunc(w, m, index, itm)
}

// default renderFunc used by the delegate if not overwritten by WithRender()
func defaultRender(w io.Writer, m list.Model, index int, itm list.Item) {
	i, ok := itm.(Entry)
	if !ok {
		return
	}

	str := fmt.Sprintf("%s%s. %s\n%s",
		colorizer.Pip(uint(index), uint(m.Index())),
		colorizer.Index(index+1),
		stylesheet.Header1Style.Render(i.Name),
		i.Details)
	fmt.Fprint(w, str)
}

//#endregion

//#region DelegateOption

// modifiers on the item delegate to change how the list displays entries
type DelegateOption func(*Delegate)

// Alter the number of lines allocated to each item.
// Height should be set equal to 1 + the lipgloss.Height of your Item.Details (1+ for Title) if
// using the default render function.
// Values above or below that can have... unpredictable... results.
func WithHeight(h int) DelegateOption {
	return func(d *Delegate) { d.entryHeight = h }
}

// Alter the number of lines between each item
func WithSpacing(s int) DelegateOption {
	return func(d *Delegate) { d.entrySpacing = s }
}

// Alter how each item is displayed in the list.
// Be careful using this option, as it cause your list to look different to others within gwcli.
func WithRender(f func(w io.Writer, m list.Model, index int, listItem list.Item)) DelegateOption {
	return func(dd *Delegate) { dd.renderFunc = f }
}

//#endregion

// #region KeyMap

// Very similar to list.DefaultKeyMap, but has the quits removed and conflicting filter keys
// reassigned.
func keyMap() list.KeyMap {
	return list.KeyMap{
		// Browsing.
		CursorUp: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		CursorDown: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		PrevPage: key.NewBinding(
			key.WithKeys("left", "h", "pgup", "b", "u"),
			key.WithHelp("←/h/pgup", "prev page"),
		),
		NextPage: key.NewBinding(
			key.WithKeys("right", "l", "pgdown", "f", "d"),
			key.WithHelp("→/l/pgdn", "next page"),
		),
		GoToStart: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("g/home", "go to start"),
		),
		GoToEnd: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("G/end", "go to end"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		ClearFilter: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "clear filter"),
		),

		// Filtering.
		CancelWhileFiltering: key.NewBinding(
			key.WithKeys("alt+/"),
			key.WithHelp("alt+/", "cancel"),
		),
		AcceptWhileFiltering: key.NewBinding(
			key.WithKeys("tab", "shift+tab", "ctrl+k", "up", "ctrl+j", "down"),
			key.WithHelp("tab", "apply filter"),
		),

		// Toggle help.
		ShowFullHelp: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "more"),
		),
		CloseFullHelp: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "close help"),
		),
	}
}

//#endregion
