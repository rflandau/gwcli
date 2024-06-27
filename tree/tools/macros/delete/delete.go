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
	"github.com/gravwell/gravwell/v3/client/types"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	use     string   = "delete"
	short   string   = "Delete a macro"
	long    string   = "Delete a macro by id or selection"
	aliases []string = []string{""}
)

// text to display when a macro would have been deleted if not for --dryrun
const dryrunDeletionText = "DRYRUN: Would have deleted macro %v(UID: %v)"

// text to display when deletion is skipped due to error
const errorNoDeleteText = "An error occured: %v.\nAbstained from deletion."

var localFS pflag.FlagSet

func NewMacroDeleteAction() action.Pair {
	cmd := treeutils.NewActionCommand(use, short, long, aliases, run)

	localFS = flags()
	cmd.Flags().AddFlagSet(&localFS)

	return treeutils.GenerateAction(cmd, Delete)
}

func run(c *cobra.Command, _ []string) {
	var (
		dryrun bool
		err    error
	)
	if dryrun, err = c.Flags().GetBool("dryrun"); err != nil {
		clilog.TeeError(c.ErrOrStderr(), fmt.Sprintf(errorNoDeleteText, err))
		return
	}

	// if an ID was given, just issue a delete
	if duid, err := c.Flags().GetUint64("uid"); err != nil {
		clilog.TeeError(c.ErrOrStderr(), fmt.Sprintf(errorNoDeleteText, err))
		return
	} else if duid != 0 {
		if dryrun { // just fetch the macro
			m, err := connection.Client.GetMacro(duid)
			if err != nil {
				clilog.TeeError(c.ErrOrStderr(), fmt.Sprintf(errorNoDeleteText, err))
				return
			}
			tea.Printf(dryrunDeletionText,
				m.Name, m.UID)
			return
		}
		if err := connection.Client.DeleteMacro(duid); err != nil {
			clilog.TeeError(c.ErrOrStderr(), fmt.Sprintf(errorNoDeleteText, err))
			return
		}
		fmt.Printf("Successfully deleted macro (UID: %v)\n", duid)
		return
	}
	// in script mode, fail out
	if script, err := c.Flags().GetBool("script"); err != nil {
		clilog.TeeError(c.ErrOrStderr(), fmt.Sprintf(errorNoDeleteText, err))
		return
	} else if script { // no id given, fail out
		clilog.TeeError(c.OutOrStdout(), "--uid is required in script mode")
		return
	}
	// TODO spin up standalone prompt selection

}

func flags() pflag.FlagSet {
	fs := pflag.FlagSet{}
	fs.Uint64("uid", 0, "macro id to delete")
	fs.Bool("dryrun", false, "skips the actual deletion")

	return fs
}

type mode uint

const (
	selecting mode = iota
	quitting
)

type delete struct {
	mode            mode
	done            bool
	list            list.Model
	listInitialized bool
	err             error
	fs              pflag.FlagSet
}

var Delete action.Model = Initial()

func Initial() *delete {
	d := &delete{mode: selecting}

	// list initialization is done in SetArgs()

	return d
}

func (d *delete) Update(msg tea.Msg) tea.Cmd {
	if len(d.list.Items()) == 0 {
		d.done = true
		return tea.Println("You have no macros that can be deleted.")
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.list.SetSize(msg.Width, msg.Height)
		return nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			// fetch the item under the cursor
			baseitm := d.list.Items()[d.list.Cursor()]
			if itm, ok := baseitm.(item); !ok {
				clilog.Writer.Warnf("failed to type assert %v as an item", baseitm)
				return tea.Printf(errorNoDeleteText+"\n", "failed type assertion")
			} else {
				d.done = true
				d.mode = quitting
				if dryrun, err := d.fs.GetBool("dryrun"); err != nil {
					clilog.Writer.Warnf("failed to fetch dryrun flag: %v", err)
					return tea.Printf(errorNoDeleteText+"\n", err)
				} else if dryrun {
					return tea.Printf(dryrunDeletionText,
						itm.Title(), itm.UID)
				} else {
					// destroy the selected macro
					if err := connection.Client.DeleteMacro(itm.ID); err != nil {
						clilog.Writer.Warnf("failed to delete macro (ID: %v): %v", itm.ID, err)
						return tea.Printf(errorNoDeleteText+"\n", err)
					}
					// remove it from the list
					d.list.RemoveItem(d.list.Cursor())
				}

				return tea.Printf("Deleted macro %v(ID: %v/UID: %v)", itm.Title(), itm.ID, itm.UID)
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
	return d.done

}

func (d *delete) Reset() error {
	d.mode = selecting
	d.done = false
	d.err = nil
	// the current state of the list is retained
	return nil
}

func (d *delete) SetArgs(_ *pflag.FlagSet, tokens []string) (invalid string, onStart []tea.Cmd, err error) {
	if !d.listInitialized { // if the this the first run, initialize the list from all macros
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

		d.listInitialized = true
	} else {
		// this could probably be optimized to directly operate just on changed records
		// rather than overwriting with a full re-sort
		// or at least by making it async with a ready-check in Update (plus timeout cancel context)
		itms, err := fetchMacroListAsItems()
		if err != nil {
			return "", nil, err
		}
		d.list.SetItems(itms)

	}
	// flagset
	d.fs = flags()
	if err := d.fs.Parse(tokens); err != nil {
		return "", nil, err
	}

	return "", nil, nil
}

// Returns all user macros as an item array ready for the list bubble
func fetchMacroListAsItems() ([]list.Item, error) {
	var items []list.Item
	ud, err := connection.Client.MyInfo()
	if err != nil {
		return nil, err
	}
	if macros, err := connection.Client.GetUserMacros(ud.UID); err != nil {
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
