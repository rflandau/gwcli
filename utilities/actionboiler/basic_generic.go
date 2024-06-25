// A basic action is the simple action: it does its thing and returns a string to be printed to the
// terminal. Give it the function you want performed when the action is invoked and have it return
// whatever string value you want printed to the screen, if at all.

package actionboiler

import (
	"fmt"
	"gwcli/action"
	"gwcli/treeutils"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Creates a new Basic action fully featured for Cobra and Mother usage.
// The given act func will be executed when the action is triggered and its result printed to the
// screen.
//
// NOTE: The tea.Cmd returned by act will be thrown away if run in a Cobra context.
func NewBasicAction(use, short, long string, aliases []string, act func() (string, tea.Cmd)) (*cobra.Command, BasicAction) {
	cmd := treeutils.NewActionCommand(
		use,
		short,
		long,
		aliases,
		func(c *cobra.Command, _ []string) {
			s, _ := act()
			fmt.Fprintf(c.OutOrStdout(), "%v\n", s)
		})

	return cmd, BasicAction{fn: act}
}

//#region interactive mode (model) implementation

type BasicAction struct {
	done bool
	fn   func() (string, tea.Cmd)
}

var _ action.Model = &BasicAction{}

func (ba *BasicAction) Update(msg tea.Msg) tea.Cmd {
	ba.done = true
	s, cmd := ba.fn()
	return tea.Sequence(tea.Println(s), cmd)
}

func (*BasicAction) View() string {
	return ""
}

func (ba *BasicAction) Done() bool {
	return ba.done
}

func (ba *BasicAction) Reset() error {
	ba.done = false
	return nil
}

func (ba *BasicAction) SetArgs(_ *pflag.FlagSet, _ []string) (_ string, _ []tea.Cmd, _ error) {
	return "", nil, nil
}

//#endregion interactive mode (model) implementation
