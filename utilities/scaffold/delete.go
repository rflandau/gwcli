package scaffold

/**
 * A delete action consumes a list of delete-able items, allowing the user to select them
 * interactively or by passing a (numeric or UUID) ID.
 *
 * Delete actions have the --dryrun and --id default flags.
 *
 * Implementations will probably look a lot like:
 *
var aliases []string = []string{}

	func NewMacroDeleteAction() action.Pair {
		return scaffold.NewDeleteAction(aliases, "macro", "macros", del,
			func() ([]scaffold.Item[uint64], error) {
				ms, err := connection.Client.GetUserGroupsMacros()
				if err != nil {
					return nil, err
				}
				slices.SortFunc(ms, func(m1, m2 types.SearchMacro) int {
					return strings.Compare(m1.Name, m2.Name)
				})
				var items = make([]scaffold.Item[uint64], len(ms))
				for i := range ms {
					items[i] = macroItem{id: ms[i].ID, name: ms[i].Name}
				}
				return items, nil
			})
	}

	func del(dryrun bool, id uint64) error {
		if dryrun {
			_, err := connection.Client.GetMacro(id)
			return err
		}

		return connection.Client.DeleteMacro(id)
	}

	type macroItem struct {
		id   uint64
		name string
	}

type macroItem struct {
	id   uint64
	name string
}

var _ scaffold.Item[uint64] = macroItem{}

func (mi macroItem) ID() uint64          { return mi.id }
func (mi macroItem) FilterValue() string { return mi.name }
func (mi macroItem) String() string      { return mi.name }
 *
*/

import (
	"fmt"
	"gwcli/action"
	"gwcli/clilog"
	"gwcli/treeutils"
	"strconv"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/gravwell/gravwell/v3/client"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/exp/constraints"
)

type id_t interface {
	constraints.Integer | uuid.UUID
}

// Returns str converted to an id of type I.
// All hail the modern Library of Alexandira (https://stackoverflow.com/a/71048872).
func FromString[I id_t](str string) (I, error) {
	var (
		err error
		ret I
	)

	switch p := any(&ret).(type) {
	case *uuid.UUID:
		var u uuid.UUID
		u, err = uuid.Parse(str)
		*p = u
	case *uint:
		var i uint64
		i, err = strconv.ParseUint(str, 10, 64)
		*p = uint(i)
	case *uint8:
		var i uint64
		i, err = strconv.ParseUint(str, 10, 8)
		*p = uint8(i)
	case *uint16:
		var i uint64
		i, err = strconv.ParseUint(str, 10, 8)
		*p = uint16(i)
	case *uint32:
		var i uint64
		i, err = strconv.ParseUint(str, 10, 8)
		*p = uint32(i)
	case *uint64:
		var i uint64
		i, err = strconv.ParseUint(str, 10, 8)
		*p = uint64(i)
	case *int:
		*p, err = strconv.Atoi(str)
	case *int8:
		var i int64
		i, err = strconv.ParseInt(str, 10, 8)
		*p = int8(i)
	case *int16:
		var i int64
		i, err = strconv.ParseInt(str, 10, 32)
		*p = int16(i)
	case *int32:
		var i int64
		i, err = strconv.ParseInt(str, 10, 32)
		*p = int32(i)
	case *int64:
		var i int64
		i, err = strconv.ParseInt(str, 10, 32)
		*p = int64(i)
	default:
		return ret, fmt.Errorf("unknown id type %#v", p)
	}
	return ret, err
}

// A function that performs the (faux-, on dryrun) deletion once an item is picked
// only returns a value if the delete (or select, on dry run) failed
type deleteFunc[I id_t] func(dryrun bool, id I) error

// A function that fetches and formats the list of delete-able items.
// It must return an array of a struct that implements the Item interface.
type fetchFunc[I id_t] func() ([]Item[I], error)

// text to display when deletion is skipped due to error
const (
	errorNoDeleteText = "An error occured: %v.\nAbstained from deletion."
	dryrunSuccessText = "DRYRUN: %v (ID %v) would have been deleted"
	deleteSuccessText = "%v (ID %v) deleted"
)

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
func NewDeleteAction[I id_t](aliases []string, singular, plural string,
	del deleteFunc[I], fch fetchFunc[I]) action.Pair {
	cmd := treeutils.NewActionCommand(
		"delete",
		"delete a "+singular,
		"delete a "+singular+" by id or selection",
		aliases,
		func(c *cobra.Command, s []string) {
			// fetch values from flags
			id, dryrun, err := fetchFlagValues[I](c.Flags())
			if err != nil {
				clilog.Tee(clilog.ERROR, c.ErrOrStderr(), err.Error())
				return
			}

			var zero I
			if id == zero {
				if _, err := c.Flags().GetBool("script"); err != nil {
					clilog.Tee(clilog.ERROR, c.ErrOrStderr(), err.Error())
					return
				} else { //else if script
					fmt.Fprintf(c.ErrOrStderr(), "--id is required in script mode")
					return
				}
				// TODO spin up mother (or independent Delete Model) if !script
			}

			if err := del(dryrun, id); err != nil {
				clilog.Tee(clilog.ERROR, c.ErrOrStderr(), err.Error())
				return
			} else if dryrun {
				fmt.Fprintf(c.OutOrStdout(), dryrunSuccessText+"\n", singular, id)
			} else {
				fmt.Fprintf(c.OutOrStdout(), deleteSuccessText+"\n",
					singular, id)
			}
		})
	fs := flags()
	cmd.Flags().AddFlagSet(&fs)
	return treeutils.GenerateAction(cmd, newDeleteModel[I](del, fch))
}

// base flagset
func flags() pflag.FlagSet {
	fs := pflag.FlagSet{}
	fs.Bool("dryrun", false, "feigns deletions, descibing actions that "+
		lipgloss.NewStyle().Italic(true).Render("would")+" have been taken")
	fs.String("id", "", "ID of the item to be deleted")
	return fs
}

// helper function for getting and casting flag values
func fetchFlagValues[I id_t](fs *pflag.FlagSet) (id I, dryrun bool, _ error) {
	if strid, err := fs.GetString("id"); err != nil {
		return id, false, err
	} else if strid != "" {
		id, err = FromString[I](strid)
		if err != nil {
			return id, dryrun, err
		}
	}
	if dr, err := fs.GetBool("dryrun"); err != nil {
		return id, dryrun, err
	} else {
		dryrun = dr
	}

	return
}

//#region interactive mode (model) implementation

type mode uint

const (
	selecting mode = iota
	quitting
)

type deleteModel[I id_t] struct {
	itemSingular string // "macro", "kit", "query"
	itemPlural   string // "macros", "kits", "queries"
	mode         mode   // current mode
	list         list.Model

	flagset pflag.FlagSet // parsed flag values (set in SetArgs)
	dryrun  bool

	df deleteFunc[I] // function to delete an item
	ff fetchFunc[I]  // function to get all delete-able items
}

func newDeleteModel[I id_t](del deleteFunc[I], fch fetchFunc[I]) *deleteModel[I] {
	d := &deleteModel[I]{mode: selecting}
	d.flagset = flags()

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
		return tea.Printf("You have no %v that can be deleted", d.itemPlural)
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
			if err := d.df(d.dryrun, itm.ID()); err != nil {
				return tea.Printf(errorNoDeleteText+"\n", err)
			}
			go d.list.RemoveItem(d.list.Index())
			if d.dryrun {
				return tea.Printf(dryrunSuccessText,
					d.itemSingular, itm.ID())
			} else {
				return tea.Printf(deleteSuccessText,
					d.itemSingular, itm.ID())
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
			return "Not deleting any " + d.itemPlural + "..."
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
	d.flagset = flags()
	// the current state of the list is retained
	return nil
}

func (d *deleteModel[I]) SetArgs(_ *pflag.FlagSet, tokens []string) (invalid string, onStart []tea.Cmd, err error) {
	var zero I
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
	d.list.Title = "Select a " + d.itemSingular + " to delete"

	d.list.SetFilteringEnabled(true)

	// disable quit keys; they clash with mother
	d.list.KeyMap.ForceQuit.SetEnabled(false)
	d.list.KeyMap.Quit.SetEnabled(false)

	// flags and flagset
	if err := d.flagset.Parse(tokens); err != nil {
		return "", nil, err
	}
	id, dryrun, err := fetchFlagValues[I](&d.flagset)
	if err != nil {
		return "", nil, err
	} else if id != zero { // if id was set, attempt to skip directly to deletion
		d.mode = quitting
		if err := d.df(dryrun, id); err != nil {
			// check for sentinel errors
			// NOTE: this relies on the client log consistently returning 404s as ClientErrors,
			// which I cannot guarentee
			if err, ok := err.(*client.ClientError); ok && err.StatusCode == 404 {
				return "", []tea.Cmd{
					tea.Printf("Did not find a valid %v with ID %v", d.itemSingular, id),
				}, nil
			}
			return "", nil, err
		} else if dryrun {
			return "",
				[]tea.Cmd{tea.Printf(dryrunSuccessText, d.itemSingular, id)},
				nil
		}
		return "",
			[]tea.Cmd{tea.Printf(deleteSuccessText, d.itemSingular, id)},
			nil

	}
	return "", nil, nil
}
