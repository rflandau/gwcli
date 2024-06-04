/**
 * The action package tests and maintains the action map, which bolts
 * subroutines onto Actions (leaves) in the cobra command tree so Mother can
 * call them interactively.
 *
 * Each Action's Model is implemented and *instantiated* in its own package
 * (ex: tree/tools/macros/macrosactions) and added to the map as part of the
 * tree's assembly.
 * When that cobra.Command is invoked interactively, Mother uses the action map
 * to find the bolted-on subroutines to supplant her own Update and View
 * subroutines until the action is `Done()`.
 * Reset() is used to clear the done status and any other no-longer-relevant
 * data so the action can be invoked again cleanly.
 */
package action

import (
	"errors"
	"gwcli/group"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

//#regions errors

var (
	ErrNotAnAction = errors.New("given command is not an action")
)

//#endregion

type Model interface {
	Update(msg tea.Msg) tea.Cmd
	View() string
	Done() bool
	Reset() error
	SetArgs([]string) (bool, error)
}

// Temp tuple used to construct the Action Map
// Associates the Action command with its bolted-on Update/View subroutines
type Pair struct {
	Action *cobra.Command
	Model  Model
}

//#region action map

/* Maps key(command) -> Model (the bolted-on Elm Arch subroutines) */
var actions = map[string]Model{}

/** GetModel returns the Model subroutines associated to the given Action. */
func GetModel(c *cobra.Command) (m Model, err error) {
	if !Is(c) {
		return nil, ErrNotAnAction
	}
	return actions[key(c)], nil
}

/** AddModel adds a new action and its subroutines to the action map */
func AddModel(c *cobra.Command, m Model) {
	//fmt.Printf("Inserting %s into action map\n", key(c))
	actions[key(c)] = m
}

/* Generates a string key from a command. Extracted for consistency */
func key(c *cobra.Command) string {
	var parentName string = "~"
	if c.Parent() != nil {
		parentName = c.Parent().Name()
	}
	return parentName + "/" + c.Name()
}

//#endregion
/**
 * Given a cobra.Command, returns whether it is an Action (and thus can supplant
 * Mother's Elm cycle) or a Nav.
 */
func Is(cmd *cobra.Command) bool {
	if cmd == nil { // sanity check
		panic("cmd cannot be nil!")
	}
	// does not `return cmd.GroupID == treeutils.ActionID` to facilitate sanity check
	switch cmd.GroupID {
	case group.ActionID:
		return true
	case group.NavID:
		return false
	default: // sanity check
		panic("cmd '" + cmd.Name() + "' is neither a nav nor an action!")
	}
}
