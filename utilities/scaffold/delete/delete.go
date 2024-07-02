package scaffolddelete

import (
	"fmt"
	"gwcli/action"
	"gwcli/clilog"
	"gwcli/treeutils"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const use = "delete"

// text to display when deletion is skipped due to error
const errorNoDeleteText = "An error occured: %v.\nAbstained from deletion."

func NewDeleteAction(short, long string, aliases []string) action.Pair {
	cmd := treeutils.NewActionCommand(use, short, long, aliases, run)
	return treeutils.GenerateAction(cmd, newDeleteModel())
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

type deleteModel struct {
	classificationPlural string // "macros", "kits", "queries"
	mode                 mode   // current mode
	list                 list.Model
	err                  error
	fs                   pflag.FlagSet
}

func newDeleteModel() *deleteModel {
	d := &deleteModel{mode: selecting}
	d.fs = flags()

	return d
}

func (d *deleteModel) Update(msg tea.Msg) tea.Cmd {
	if d.Done() {
		return nil
	}
	if len(d.list.Items()) == 0 {
		d.mode = quitting
		return tea.Printf("You have no %v that can be deleted", d.classificationPlural)
	}

	// TODO
	return nil

}

func (d *deleteModel) View() string {
	switch d.mode {
	case quitting:
		// This is unlikely to ever be shown before Mother reasserts control and wipes it
		itm := d.list.SelectedItem()
		if itm == nil {
			return "Not deleting any " + d.classificationPlural + "..."
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

func (d *deleteModel) Done() bool {
	return d.mode == quitting
}

func (d *deleteModel) Reset() error {
	d.mode = selecting
	d.err = nil
	d.fs = flags()
	// the current state of the list is retained
	return nil
}

func (d *deleteModel) SetArgs(_ *pflag.FlagSet, tokens []string) (invalid string, onStart []tea.Cmd, err error) {
	// initialize the list
	/*d.list = list.New([]list.Item{}, itemDelegate{}, 80, 20)
	d.list.Title = "Select a macro to delete"

	//itms, err := fetchMacroListAsItems()
	if err != nil {
		return "", nil, err
	}
	d.list.SetItems(itms)
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
	}*/
	return "", nil, nil
}
