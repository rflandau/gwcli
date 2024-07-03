package scaffold

import (
	"fmt"
	"gwcli/action"
	"gwcli/clilog"
	"gwcli/treeutils"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/exp/constraints"
)

type id_t interface {
	constraints.Integer | uuid.UUID
}

// A function that performs the (faux-, on dryrun) deletion once an item is picked
// only returns a value if the delete (or select, on dry run) failed
type deleteFunc[I id_t] func(dryrun bool, id I) error

// A function that fetches and formats the list of delete-able items.
// It must return an array of a struct that implements the Item interface.
type fetchFunc[I id_t] func() ([]Item[I], error)

// text to display when deletion is skipped due to error
const errorNoDeleteText = "An error occured: %v.\nAbstained from deletion."

// NewDeleteAction creates and returns a cobra.Command suitable for use as a delete action.
// Base flags:
//
//	--dryrun (SELECT, as a mock deletion),
//
//	--id (immediately attempt deletion on the given id)
//
// You must provide two functions to instantiate a generic delete:
//
// Del is a function that performs the actual (mock) deletion.
// It is given the dryrun boolean and an ID value and returns an error only if the delete or select
// failed.
//
// Fch is a function that fetches all, delete-able records for the user to pick from.
// It returns a user-defined struct fitting the Item interface.
func NewDeleteAction[I id_t](short, long string, aliases []string, singular, plural string,
	del deleteFunc[I], fch fetchFunc[I]) action.Pair {
	cmd := treeutils.NewActionCommand("delete", short, long, aliases, run)
	return treeutils.GenerateAction(cmd, newDeleteModel[I](del, fch))
}

func run(*cobra.Command, []string) {
	// TODO
}

// base flagset
func flags() pflag.FlagSet {
	fs := pflag.FlagSet{}
	fs.Bool("dryrun", false, "feigns deletions, descibing actions that "+
		lipgloss.NewStyle().Italic(true).Render("would")+" have been taken")
	// TODO implement --id
	return fs
}

//#region interactive mode (model) implementation

type mode uint

const (
	selecting mode = iota
	quitting
)

type deleteModel[I id_t] struct {
	classificationSingular string // "macro", "kit", "query"
	classificationPlural   string // "macros", "kits", "queries"
	mode                   mode   // current mode
	list                   list.Model
	flags                  struct { // parsed flag values (set in SetArgs)
		set    pflag.FlagSet
		dryrun bool
	}
	df deleteFunc[I] // function to delete an item
	ff fetchFunc[I]  // function to get all delete-able items
}

func newDeleteModel[I id_t](del deleteFunc[I], fch fetchFunc[I]) *deleteModel[I] {
	d := &deleteModel[I]{mode: selecting}
	d.flags.set = flags()
	d.df = del
	d.ff = fch

	return d
}

func (d *deleteModel[I]) Update(msg tea.Msg) tea.Cmd {
	if d.Done() {
		return nil
	}
	if len(d.list.Items()) == 0 {
		d.mode = quitting
		return tea.Printf("You have no %v that can be deleted", d.classificationPlural)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.list.SetSize(msg.Width, msg.Height)
		return nil
	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter {
			var (
				baseitm list.Item // item stored in the list
				itm     Item[I]   // baseitm cast to our expanded item type
				ok      bool      // type assertion result
			)
			baseitm = d.list.Items()[d.list.Index()]
			if itm, ok = baseitm.(Item[I]); !ok {
				clilog.Writer.Warnf("failed to type assert %#v as an item", baseitm)
				return tea.Printf(errorNoDeleteText+"\n", "failed type assertion")
			}
			d.mode = quitting

			// attempt to delete the item
			if err := d.df(d.flags.dryrun, itm.ID()); err != nil {
				return tea.Printf(errorNoDeleteText+"\n", err)
			}
			go d.list.RemoveItem(d.list.Index())
			if d.flags.dryrun {
				return tea.Printf("DRYRUN: %v (ID %v) would have been deleted",
					d.classificationSingular, itm.ID())
			} else {
				return tea.Printf("%v (ID %v) deleted",
					d.classificationSingular, itm.ID())
			}
		}
	}

	var cmd tea.Cmd
	d.list, cmd = d.list.Update(msg)

	return cmd

}

func (d *deleteModel[I]) View() string {
	switch d.mode {
	case quitting:
		// This is unlikely to ever be shown before Mother reasserts control and wipes it
		itm := d.list.SelectedItem()
		if itm == nil {
			return "Not deleting any " + d.classificationPlural + "..."
		}
		if searchitm, ok := itm.(Item[I]); !ok {
			clilog.Writer.Warnf("Failed to type assert selected %v", itm)
			return "An error has occurred. Exitting..."
		} else {
			return fmt.Sprintf("Deleting %v...\n", searchitm.String())
		}
	case selecting:
		return "\n" + d.list.View()
	default:
		clilog.Writer.Warnf("Unknown mode %v", d.mode)
		return "An error has occurred. Exitting..."
	}
}

func (d *deleteModel[I]) Done() bool {
	return d.mode == quitting
}

func (d *deleteModel[I]) Reset() error {
	d.mode = selecting
	d.flags.set = flags()
	// the current state of the list is retained
	return nil
}

func (d *deleteModel[I]) SetArgs(_ *pflag.FlagSet, tokens []string) (invalid string, onStart []tea.Cmd, err error) {
	// initialize the list
	itms, err := d.ff()
	if err != nil {
		return "", nil, err
	}
	// while Item[I] satisfies the list.Item interface, Go will not implicitly
	// convert []Item[I] -> []list.Item
	// remember to assert these items as Item[I] on use
	// TODO do we hide this in here, at the cost of an extra n? Or move it out to ff?
	simpleitems := make([]list.Item, len(itms))
	for i := range itms {
		simpleitems[i] = itms[i]
	}

	d.list = list.New(simpleitems, itemDelegate[I]{}, 80, 20)
	d.list.Title = "Select a " + d.classificationSingular + " to delete"

	d.list.SetFilteringEnabled(true)

	// disable quit keys; they clash with mother
	d.list.KeyMap.ForceQuit.SetEnabled(false)
	d.list.KeyMap.Quit.SetEnabled(false)

	// flags and flagset
	if err := d.flags.set.Parse(tokens); err != nil {
		return "", nil, err
	}
	if d.flags.dryrun, err = d.flags.set.GetBool("dryrun"); err != nil {
		return "", nil, err
	}

	// if --id was given attempt to act and quit immediately
	/*if id, err := d.fs.GetUint64("id"); err != nil {
		return "", nil, err
	} else if id != 0 {
		d.mode = quitting
		//dr, err := deleteMacro(&d.fs, id)
		if err != nil {
			// check for sentinel errors
			if err, ok := err.(*client.ClientError); ok && err.StatusCode == 404 {
				return "", []tea.Cmd{tea.Printf("Did not find a valid macro with ID %v", id)}, nil
			}

			return "", nil, err
		} else if dr {
			return "",
				[]tea.Cmd{tea.Printf("DRYRUN: Macro (UID: %v) would have been deleted\n", id)},
				nil
		}
		return "",
			[]tea.Cmd{tea.Printf("Deleted macro (UID: %v)\n", id)},
			nil
	} */
	return "", nil, nil
}
