/**
 * Actors are child-Actions of Mother that implement the Model-Update-View
 * architecture.
 * Each Actor is implemented and *instantiated* in its own package, then
 * associated to an cobra.Command in the command tree.
 * When that cobra.Command is invoked interactively, Mother calls up the Actor
 * to supplant her own Update and View subroutines until the Actor is .Done().
 * Reset() is used to clear the done status and any other no-longer-relevant
 * data so the action can be invoked again cleanly.
 */
package action

import (
	"errors"
	"fmt"
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
}

// Temp tuple used to construct the Action Map
// Associates the Action command with its bolted-on Update/View subroutines
type Pair struct {
	Action *cobra.Command
	Model  Model
}

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
	fmt.Printf("Inserting %s into action map\n", key(c))
	actions[key(c)] = m
}

/* Generates a string key from a command. Extracted for consistency */
func key(c *cobra.Command) string {
	return c.Parent().Name() + "/" + c.Name()
}

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
