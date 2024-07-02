package scaffold

import (
	"fmt"
	"gwcli/action"
	"gwcli/clilog"
	"gwcli/treeutils"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/gravwell/gravwell/v3/client"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/exp/constraints"
)

type id_t interface {
	constraints.Integer | uuid.UUID
}

// a given function that performs the (faux-, on dryrun) deletion once an item is picked
// only returns a value if the delete (or select, on dry run) failed
type deleteFunc[I id_t] func(dryrun bool, id I) error

// a given function that fetches and formats the list of delete-able items
// TODO how do we allow this to return specifically []Item
type fetchFunc[I id_t] func() ([]Item[I], error)

// text to display when deletion is skipped due to error
const errorNoDeleteText = "An error occured: %v.\nAbstained from deletion."

func NewDeleteAction[I id_t](short, long string, aliases []string, singular, plural string,
	del deleteFunc[I], fch fetchFunc[I]) action.Pair {
	cmd := treeutils.NewActionCommand("delete", short, long, aliases, run)
	return treeutils.GenerateAction(cmd, newDeleteModel[I](del, fch))
}

func run(*cobra.Command, []string) {

}

func flags() pflag.FlagSet {
	fs := pflag.FlagSet{}

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
	err                    error
	fs                     pflag.FlagSet
	df                     deleteFunc[I] // function to delete an item
	ff                     fetchFunc[I]  // function to get all delete-able items
}

func newDeleteModel[I id_t](df deleteFunc[I], ff fetchFunc[I]) *deleteModel[I] {
	d := &deleteModel[I]{mode: selecting}
	d.fs = flags()
	d.df = df
	d.ff = ff

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
		switch msg.Type {
		case tea.KeyEnter:
			baseitm := d.list.Items()[d.list.Index()]
			if itm, ok := baseitm.(Item[I]); !ok {
				clilog.Writer.Warnf("failed to type assert %#v as an item", baseitm)
				return tea.Printf(errorNoDeleteText+"\n", "failed type assertion")
			} else {
				d.mode = quitting

				// TODO get dryrun mode
				// attempt to delete the item
				err := d.df(false, itm.ID())

			}

			// attempt to delete the function
		}
	}

	// TODO
	return nil

}

func (d *deleteModel[I]) View() string {
	switch d.mode {
	case quitting:
		// This is unlikely to ever be shown before Mother reasserts control and wipes it
		itm := d.list.SelectedItem()
		if itm == nil {
			return "Not deleting any " + d.classificationPlural + "..."
		}
		if searchitm, ok := itm.(S); !ok {
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
	d.err = nil
	d.fs = flags()
	// the current state of the list is retained
	return nil
}

func (d *deleteModel[I]) SetArgs(_ *pflag.FlagSet, tokens []string) (invalid string, onStart []tea.Cmd, err error) {
	// initialize the list
	itms, err := d.ff()
	if err != nil {
		return "", nil, err
	}
	// TODO Item[I] satisfies the list.Item interface; why is this unacceptable?
	d.list = list.New(itms, itemDelegate{}, 80, 20)
	d.list.Title = "Select a " + d.classificationSingular + " to delete"

	d.list.SetFilteringEnabled(true)

	// disable quit keys; they clash with mother
	d.list.KeyMap.ForceQuit.SetEnabled(false)
	d.list.KeyMap.Quit.SetEnabled(false)

	// flagset
	if err := d.fs.Parse(tokens); err != nil {
		return "", nil, err
	}

	// if --id was given attempt to act and quit immediately
	if id, err := d.fs.GetUint64("id"); err != nil {
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
	}
	return "", nil, nil
}
