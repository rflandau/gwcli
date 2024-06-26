// A basic action is the simplest action: it does its thing and returns a string to be printed to the
// terminal. Give it the function you want performed when the action is invoked and have it return
// whatever string value you want printed to the screen, if at all.

package scaffold

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
func NewBasicAction(use, short, long string, aliases []string,
	act func(*pflag.FlagSet) (string, tea.Cmd), addtlFlags *pflag.FlagSet) action.Pair {

	cmd := treeutils.NewActionCommand(
		use,
		short,
		long,
		aliases,
		func(c *cobra.Command, _ []string) {
			s, _ := act(c.Flags())
			fmt.Fprintf(c.OutOrStdout(), "%v\n", s)
		})

	if addtlFlags != nil {
		cmd.Flags().AddFlagSet(addtlFlags)
	}

	return treeutils.GenerateAction(cmd, &BasicAction{fs: *cmd.Flags(), baseFS: *cmd.Flags(), fn: act})
}

//#region interactive mode (model) implementation

type BasicAction struct {
	done   bool
	fs     pflag.FlagSet
	baseFS pflag.FlagSet // the flagset to restore to
	fn     func(*pflag.FlagSet) (string, tea.Cmd)
}

var _ action.Model = &BasicAction{}

func (ba *BasicAction) Update(msg tea.Msg) tea.Cmd {
	ba.done = true
	s, cmd := ba.fn(&ba.fs)
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
	ba.fs = ba.baseFS
	return nil
}

func (ba *BasicAction) SetArgs(_ *pflag.FlagSet, tokens []string) (_ string, _ []tea.Cmd, err error) {
	// we must parse manually each interactive call, as we restore fs from base each invocation
	err = ba.fs.Parse(tokens)
	if err != nil {
		return "", nil, err
	}
	return "", nil, nil
}

//#endregion interactive mode (model) implementation
