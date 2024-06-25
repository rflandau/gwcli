/**
 * Builtins are special, meta actions users can invoke from Mother's prompt, no
 * matter their pwd.
 */
package mother

import (
	"gwcli/clilog"
	"gwcli/stylesheet"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// invocation string -> function to be invoked
var builtins map[string](func(*Mother, []string) tea.Cmd)

// invocation string -> help string displayed when `help <builtin>` is called
var builtinHelp map[string]string

// initialize the maps used for builtin actions
func initBuiltins() {
	builtins = map[string](func(*Mother, []string) tea.Cmd){
		"help":    ContextHelp,
		"history": ListHistory,
		"quit":    quit,
		"exit":    quit}

	builtinHelp = map[string]string{
		"help": "Display context-sensitive help. Equivalent to pressing F1.\n" +
			"Calling `help` bare provides currently available navigations.\n" +
			"Help can also be passed a path to display help on remote directories or actions.\n" +
			"Ex: `help .. kits list`",
		"history": "List previous commands. Navigate history via ↑/↓",
		"quit":    "Kill the application",
		"exit":    "Kill the application",
	}
}

// Built-in, interactive help invocation
func ContextHelp(m *Mother, args []string) tea.Cmd {
	if len(args) == 0 {
		return TeaCmdContextHelp(m.pwd)
	}

	// walk the command tree
	// action or nav, print help about it
	// if invalid/no destination, print error
	wr := walk(m.pwd, args, make([]tea.Cmd, 1))

	if wr.errString != "" { // erroneous input
		return tea.Println(stylesheet.ErrStyle.Render(wr.errString))
	}
	switch wr.status {
	case foundNav, foundAction:
		return TeaCmdContextHelp(wr.endCommand)
	case foundBuiltin:
		if _, ok := builtins[args[0]]; ok {
			str, found := builtinHelp[args[0]]
			if !found {
				str = "no help defined for '" + args[0] + "'"
			}

			return tea.Printf(str)
		}

	}

	clilog.Writer.Debugf("Doing nothing (%#v)", wr)

	return nil
}

func ListHistory(m *Mother, _ []string) tea.Cmd {
	toPrint := strings.Builder{}
	rs := m.history.getAllRecords()

	// print the oldest record first, so newest record is directly over prompt
	for i := len(rs) - 1; i > 0; i-- {
		toPrint.WriteString(rs[i] + "\n")
	}

	// chomp last newline
	return tea.Println(strings.TrimSpace(toPrint.String()))
}

func quit(*Mother, []string) tea.Cmd {
	return tea.Sequence(tea.Println("Bye"), tea.Quit)
}
