/**
 * Walk is the beefy boy that enables dynamic path-finding through the tree.
 * It recusively walks a series of tokens, determining what to do at each step
 * until an acceptable endpoint is reached
 * (e.g. an executable action, a nav, an error).
 * It is both used directly for Mother traversal of the command tree as well as
 * determining the validity of a proposed path.
 */
package mother

import (
	"fmt"
	"gwcli/action"
	"gwcli/clilog"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

type walkStatus int

const (
	invalidCommand walkStatus = iota
	foundNav
	foundAction
	foundBuiltin
	erroring
)

type walkResult struct {
	endCommand     *cobra.Command // the relevent command walk completed on
	status         walkStatus     // ending state
	onCompleteCmds []tea.Cmd      // ordered list of commands to pass to the bubble tea driver
	errString      string

	// builtin function information, if relevant (else Zero vals)
	builtinStr  string
	builtinFunc func(*Mother, []string) tea.Cmd

	// contains args for actions
	remainingTokens []string // any tokens remaining for later processing by walk caller
}

// Recursively walk the tokens of the exploded user input until we run out or
// find a valid destination.
// Returns the relevant command (ending Nav destination or action to invoke),
// the type of the command (action, nav, invalid), a list of commands to pass to
// tea, and an error (if one occurred).
func walk(dir *cobra.Command, tokens []string, onCompleteCmds []tea.Cmd) walkResult {
	if len(tokens) == 0 {
		// only move if the final command was a nav
		return walkResult{
			endCommand:     dir,
			status:         foundNav,
			onCompleteCmds: onCompleteCmds,
		}
	}

	curToken := strings.TrimSpace(tokens[0])
	// if there is no token, just keep walking
	if curToken == "" {
		return walk(dir, tokens[1:], onCompleteCmds)
	}

	if bif, ok := builtins[curToken]; ok { // check for built-in command
		return walkResult{
			endCommand:      nil,
			status:          foundBuiltin,
			onCompleteCmds:  onCompleteCmds,
			builtinStr:      curToken,
			builtinFunc:     bif,
			remainingTokens: tokens[1:],
		}
	}

	if curToken == ".." { // navigate upward
		dir = up(dir)
		return walk(dir, tokens[1:], onCompleteCmds)
	}

	// test for a local command
	var invocation *cobra.Command = nil
	for _, c := range dir.Commands() {

		if c.Name() == curToken { // check name
			invocation = c
			clilog.Writer.Debugf("Match, invoking %s", invocation.Name())
		} else { // check aliases
			for _, alias := range c.Aliases {
				if alias == curToken {
					invocation = c
					clilog.Writer.Debugf("Alias match, invoking %s", invocation.Name())
					break
				}
			}
		}
		if invocation != nil {
			break
		}
	}

	// check if we found a match
	if invocation == nil {
		// user request unhandlable
		return walkResult{
			endCommand:      nil,
			status:          invalidCommand,
			onCompleteCmds:  onCompleteCmds,
			errString:       fmt.Sprintf("unknown command '%s'. Press F1 or type 'help' for relevant commands.", curToken),
			remainingTokens: tokens[1:],
		}
	}

	// split on action or nav
	if action.Is(invocation) {
		return walkResult{
			endCommand:      invocation,
			status:          foundAction,
			onCompleteCmds:  onCompleteCmds,
			remainingTokens: tokens[1:],
		}
	} else { // nav
		// navigate given path
		dir = invocation
		return walk(dir, tokens[1:], onCompleteCmds)
	}
}
