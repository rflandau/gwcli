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

var localFS pflag.FlagSet

func NewMacroDeleteAction() action.Pair {
	cmd := treeutils.NewActionCommand(use, short, long, aliases,
		func(c *cobra.Command, s []string) {
			// if an ID was given, just issue a delete
			if did, err := c.Flags().GetUint64("id"); err != nil {
				clilog.TeeError(c.ErrOrStderr(), err.Error())
				return
			} else if did != 0 {
				if dryrun, err := c.Flags().GetBool("dryrun"); err != nil {
					clilog.TeeError(c.ErrOrStderr(), err.Error())
					return
				} else if dryrun {
					// just fetch the macro
					if m, err := connection.Client.GetMacro(did); err != nil {
						clilog.TeeError(c.ErrOrStderr(), err.Error())
						return
					} else {
						tea.Printf("DRYRUN: Would have deleted macro %v(UID: %v)",
							m.Name, m.UID)
					}
				}
				if err := connection.Client.DeleteMacro(did); err != nil {
					clilog.TeeError(c.ErrOrStderr(), err.Error())
					return
				}
				fmt.Printf("Successfully deleted macro #%v\n", did)
				return
			}
			// in script mode, fail out
			if script, err := c.Flags().GetBool("script"); err != nil {
				clilog.TeeError(c.ErrOrStderr(), err.Error())
				return
			} else if script {
				// fail out
				fmt.Fprint(c.OutOrStdout(), "--id is required in script mode")
				return
			}
			// TODO spin up standalone prompt selection
		})

	localFS = flags()
	cmd.Flags().AddFlagSet(&localFS)

	return treeutils.GenerateAction(cmd, Delete)
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

	// TODO modify key map

	// list initialization is done in SetArgs()

	return d
}

func (d *delete) Update(msg tea.Msg) tea.Cmd {
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
				return tea.Println("An error occured.\nAbstained from deletion.")
			} else {
				d.done = true
				d.mode = quitting
				if dryrun, err := d.fs.GetBool("dryrun"); err != nil {
					clilog.Writer.Warnf("failed to fetch dryrun flag: %v", err)
					return tea.Println("An error occured.\nAbstained from deletion.")
				} else if dryrun {
					return tea.Printf("DRYRUN: Would have deleted macro %v(UID: %v)",
						itm.Title(), itm.UID)
				} else {
					// destroy the selected macro
					if err := connection.Client.DeleteMacro(itm.ID); err != nil {
						clilog.Writer.Warnf("failed to delete macro (ID: %v): %v", itm.ID, err)
						return tea.Println("An error occured.\nAbstained from deletion.")
					}
					// remove it from the list
					d.list.RemoveItem(d.list.Cursor())
				}

				return tea.Printf("Deleted macro %v(UID: %v)", itm.Title(), itm.UID)
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
			return fmt.Sprintf("Deleting %v (UID: %v)...\n", searchitm.Name, searchitm.UID)
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
	// if the this the first run, initialize the list from all macros
	if !d.listInitialized {
		d.list = list.New([]list.Item{}, itemDelegate{}, 80, 20)
		d.list.Title = "Select a Macro to delete"

		var items []list.Item
		ud, err := connection.Client.MyInfo()
		if err != nil {
			return "", nil, err
		}
		if macros, err := connection.Client.GetUserMacros(ud.UID); err != nil {
			return "", nil, err
		} else {
			items = make([]list.Item, len(macros))
			clilog.Writer.Debugf("macros: %v", macros)
			slices.SortFunc(macros, func(m1, m2 types.SearchMacro) int {
				return strings.Compare(m1.Name, m2.Name)
			})
			for i := range macros {
				items[i] = item(macros[i])
			}
		}
		clilog.Writer.Debugf("Setting %d items", len(items))
		d.list.SetItems(items)
		d.list.SetFilteringEnabled(false)

		d.listInitialized = true
	}

	// otherwise, compare the list for differences with the current set of macros
	// TODO
	// TODO async
	// when setting args, fetch the current set of macros and asyncronously add them to the list
	/*go func() {
		macros, err := connection.Client.GetUserMacros(connection.Client.MyUID())
		if err != nil {
			// TODO lock error before setting it
			d.err = err
			return
		}
		items := d.list.Items()

		// TODO may need to queue returned cmd
		d.list.SetItems(merge(macros, items))
	}()*/

	// flagset
	d.fs = flags()
	if err := d.fs.Parse(tokens); err != nil {
		return "", nil, err
	}

	return "", nil, nil
}
