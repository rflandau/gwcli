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
package actor

import (
	tea "github.com/charmbracelet/bubbletea"
)

type Actor interface {
	Update(msg tea.Msg) tea.Cmd
	View() string
	Done() bool
	Reset() error
}
