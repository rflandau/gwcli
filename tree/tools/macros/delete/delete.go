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
}

var Delete action.Model = Initial()

func Initial() *delete {
	d := &delete{mode: selecting}

	// list initialization is done in SetArgs()

	return d
}

func (d *delete) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.list.SetWidth(msg.Width)
		return nil
	case tea.KeyMsg:

	}

	var cmd tea.Cmd
	d.list, cmd = d.list.Update(msg)

	return cmd
}

func (d *delete) View() string {
	switch d.mode {
	case quitting:
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
		d.list = list.New([]list.Item{}, list.DefaultDelegate{}, 80, 20)
		d.list.Title = "Select a Macro to delete"

		var items []list.Item
		if macros, err := connection.Client.GetUserMacros(connection.Client.MyUID()); err != nil {
			return "", nil, err
		} else {
			items = make([]list.Item, len(macros))
			slices.SortFunc(macros, func(m1, m2 types.SearchMacro) int {
				return strings.Compare(m1.Name, m2.Name)
			})
			for i := range macros {
				items[i] = item(macros[i])
			}
		}

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

	return "", nil, nil
}
