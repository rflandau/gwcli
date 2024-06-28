// Macro deletion action.
// Displays a list of all available macros that the user can pick from in interactive mode.
// Macros are sorted by name in the list itself.
// Note that list initialization and updating occurs in SetArgs.
// This is allow lazy-processing; do not want to add startup time when we do not know a user will
// invoke this action. Similarly, we cannot guarentee that the server connection will be established
// yet (in fact, it likely would not be if we prepared the list in Initial()).
package delete

import (
	"fmt"
	"gwcli/action"
	"gwcli/clilog"
	"gwcli/connection"
	"gwcli/treeutils"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gravwell/gravwell/v3/client"
	"github.com/gravwell/gravwell/v3/client/types"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	use     string   = "delete"
	short   string   = "Delete a macro"
	long    string   = "Delete a macro by id or selection"
	aliases []string = []string{}
)

// text to display when deletion is skipped due to error
const errorNoDeleteText = "An error occured: %v.\nAbstained from deletion."

func NewMacroDeleteAction() action.Pair {
	cmd := treeutils.NewActionCommand(use, short, long, aliases, run)

	fs := flags()
	cmd.Flags().AddFlagSet(&fs)

	return treeutils.GenerateAction(cmd, NewDelete())
}

func run(c *cobra.Command, _ []string) {
	// if an ID was given, just issue a delete
	if did, err := c.Flags().GetUint64("id"); err != nil {
		clilog.TeeError(c.ErrOrStderr(), fmt.Sprintf(errorNoDeleteText, err))
		return
	} else if did != 0 {
		if dr, err := deleteMacro(c.Flags(), did); err != nil {
			clilog.TeeError(c.ErrOrStderr(),
				fmt.Sprintf("failed to delete macro (UID: %v): %v", did, err))
			return
		} else if dr {
			fmt.Fprintf(c.OutOrStdout(), "DRYRUN: Macro (UID: %v) would have been deleted\n", did)
			return
		}
		fmt.Fprintf(c.OutOrStdout(), "Deleted macro (UID: %v).\n", did)
	}
	// in script mode, fail out
	if script, err := c.Flags().GetBool("script"); err != nil {
		clilog.TeeError(c.ErrOrStderr(), fmt.Sprintf(errorNoDeleteText, err))
		return
	} else if script { // no id given, fail out
		clilog.TeeError(c.OutOrStdout(), "--id is required in script mode")
		return
	}
}

func flags() pflag.FlagSet {
	fs := pflag.FlagSet{}
	fs.Uint64("id", 0, "macro id to delete")
	fs.Bool("dryrun", false, "skips the actual deletion")

	return fs
}

type mode uint

const (
	selecting mode = iota
	quitting
)

type delete struct {
	mode mode
	list list.Model
	err  error
	fs   pflag.FlagSet
}

func NewDelete() *delete {
	d := &delete{mode: selecting}

	d.fs = flags()
	// list initialization is done in SetArgs()

	return d
}

func (d *delete) Update(msg tea.Msg) tea.Cmd {
	if d.mode == quitting {
		return nil
	}
	if len(d.list.Items()) == 0 {
		d.mode = quitting
		return tea.Println("You have no macros that can be deleted.")
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.list.SetSize(msg.Width, msg.Height)
		return nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			// fetch the item under the cursor (using the index relative to the slice, not the page)
			baseitm := d.list.Items()[d.list.Index()]
			if itm, ok := baseitm.(item); !ok {
				clilog.Writer.Warnf("failed to type assert %v as an item", baseitm)
				return tea.Printf(errorNoDeleteText+"\n", "failed type assertion")
			} else {
				d.mode = quitting
				if dr, err := deleteMacro(&d.fs, itm.ID); err != nil {
					clilog.Writer.Errorf("failed to delete macro %v (ID: %v/UID: %v): %v",
						itm.Name, itm.ID, itm.UID, err)
					return tea.Printf(errorNoDeleteText, err)
				} else if dr {
					return tea.Printf("DRYRUN: Macro %v (ID: %v/UID: %v) would have been deleted",
						itm.Name, itm.ID, itm.UID)
				}
				// remove it from the list
				d.list.RemoveItem(d.list.Index())

				return tea.Printf("Deleted macro %v(ID: %v/UID: %v)", itm.Name, itm.ID, itm.UID)
			}
		}
	}

	var cmd tea.Cmd
	d.list, cmd = d.list.Update(msg)

	return cmd
}

func (d *delete) View() string {
	switch d.mode {
	case quitting:
		// This is unlikely to ever be shown before Mother reasserts control and wipes it
		itm := d.list.SelectedItem()
		if itm == nil {
			return "Not deleting any macros..."
		}
		if searchitm, ok := itm.(item); !ok {
			clilog.Writer.Warnf("Failed to assert selected item as SearchMacro (%v)", itm)
			return "An error has occurred. Exitting..."
		} else {
			return fmt.Sprintf("Deleting %v (ID: %v/UID: %v)...\n",
				searchitm.Name, searchitm.ID, searchitm.UID)
		}
	case selecting:
		return "\n" + d.list.View()
	default:
		clilog.Writer.Warnf("Unknown mode %v", d.mode)
		return "An error has occurred. Exitting..."
	}
}

func (d *delete) Done() bool {
	return d.mode == quitting
}

func (d *delete) Reset() error {
	d.mode = selecting
	d.err = nil
	d.fs = flags()
	// the current state of the list is retained
	return nil
}

func (d *delete) SetArgs(_ *pflag.FlagSet, tokens []string) (invalid string, onStart []tea.Cmd, err error) {
	// initialize the list
	d.list = list.New([]list.Item{}, itemDelegate{}, 80, 20)
	d.list.Title = "Select a macro to delete"

	itms, err := fetchMacroListAsItems()
	if err != nil {
		return "", nil, err
	}
	d.list.SetItems(itms)
	d.list.SetFilteringEnabled(false)

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
		dr, err := deleteMacro(&d.fs, id)
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

// Deletes (or feigns deletion, if dryrun) the macro associated to the given ID.
func deleteMacro(fs *pflag.FlagSet, macroID uint64) (dryrun bool, err error) {
	if dryrun, err := fs.GetBool("dryrun"); err != nil {
		return false, err
	} else if dryrun { // fetch the macro to check existence
		_, err := connection.Client.GetMacro(macroID)
		if err != nil {
			return true, err
		}
		return true, nil
	}
	// destroy the selected macro
	if err := connection.Client.DeleteMacro(macroID); err != nil {
		return false, err
	}
	return false, nil
}

// Returns all user macros as an item array ready for the list bubble
func fetchMacroListAsItems() ([]list.Item, error) {
	var items []list.Item
	myinfo, err := connection.Client.MyInfo()
	if err != nil {
		return nil, err
	}
	if macros, err := connection.Client.GetUserMacros(myinfo.UID); err != nil {
		return nil, err
	} else {
		items = make([]list.Item, len(macros))
		slices.SortFunc(macros, func(m1, m2 types.SearchMacro) int {
			return strings.Compare(m1.Name, m2.Name)
		})
		for i := range macros {
			items[i] = item(macros[i])
		}
	}

	return items, nil
}
