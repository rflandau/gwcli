/**
 * A basic action that simply displays the command structure of gwcli using the lipgloss tree
 * functionality.
 */
package tree

import (
	"gwcli/action"
	"gwcli/group"
	"gwcli/utilities/scaffold"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss/tree"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	use   string = "tree"
	short string = "display all commands as a tree"
	long  string = "Displays a directory-tree showing the full structure of gwcli and all" +
		"available actions"
	aliases []string = []string{}
)

func NewTreeAction() action.Pair {
	return scaffold.NewBasicAction(use, short, long, aliases,
		func(c *cobra.Command, _ *pflag.FlagSet) (string, tea.Cmd) {
			lgt := tree.New()
			lgt.Root("gwcli")
			root := c.Root()
			// traverse down the command tree, setting each nav as a new sub tree and each action
			//	as a leaf
			for _, child := range root.Commands() {
				switch child.GroupID {
				case group.ActionID:
					lgt.Child(child.Name())
				case group.NavID:
					lgt.Child(walkBranch(child))
				default:
					lgt.Child(child.Name())
				}
			}

			return lgt.String(), nil
		}, nil)
}

func walkBranch(nav *cobra.Command) *tree.Tree {
	// generate a new tree, stemming from the given nav
	branchRoot := tree.New()
	branchRoot.Root(nav.Name())

	// add children of this nav to its tree
	for _, child := range nav.Commands() {
		switch child.GroupID {
		case group.ActionID:
			branchRoot.Child(child.Name())
		case group.NavID:
			branchRoot.Child(walkBranch(child))
		default:
			branchRoot.Child(child.Name())
		}
	}

	return branchRoot

}
