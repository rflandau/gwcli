/*
This package supplies the item delegate used for list bubble implementations (primarily in the
scaffolds) within gwcli. By sharing a single definition, we can ensure lists look and function
identically no matter what action or scaffold is invoking it.
*/
package listsupport

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
)

func NewList(items []list.Item, width, height int, singular, plural string) list.Model {
	// update the styles on the default delegate to wrap properly
	dlg := list.NewDefaultDelegate()
	//	dlg.Styles.NormalDesc = dlg.Styles.NormalDesc.Width(width)
	//	dlg.Styles.DimmedDesc = dlg.Styles.DimmedDesc.Width(width)
	dlg.Styles.SelectedDesc = dlg.Styles.SelectedDesc.Width(width)
	dlg.Styles.SelectedTitle = dlg.Styles.SelectedTitle.Width(width)

	l := list.New(items, dlg, 200, height)
	l.KeyMap = keyMap()
	l.SetSpinner(spinner.Moon)
	l.SetStatusBarItemName(singular, plural)
	l.SetShowTitle(false)
	return l
}

// An entry is an actual item in the list. When an item is retrieved from a list.Model, it should b
// cast to an Entry.
type Entry struct {
	title       string // the unique name of this item
	description string // the second (and potentially more) line of this item
}

func NewEntry(title, description string) Entry {
	return Entry{title: title, description: description}
}

func (e Entry) Title() string       { return e.title }
func (e Entry) Description() string { return e.description }
func (e Entry) FilterValue() string { return e.title }

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
