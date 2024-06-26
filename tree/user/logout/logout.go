// A simple logout action that logs out the current user and ends the program
package logout

import (
	"gwcli/action"
	"gwcli/connection"
	"gwcli/utilities/scaffold"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/pflag"
)

var (
	use     string   = "logout"
	short   string   = "Logout and end the session"
	long    string   = "Ends your current session and invalids your login token."
	aliases []string = []string{}
)

func NewLogoutAction() action.Pair {
	return scaffold.NewBasicAction(use, short, long, aliases, func(*pflag.FlagSet) (string, tea.Cmd) {
		connection.Client.Logout()
		connection.End()

		return "Successfully logged out", tea.Quit
	}, nil)
}
