// A simple logout action that logs out the current user and ends the program
package logout

import (
	"gwcli/action"
	"gwcli/connection"
	"gwcli/treeutils"
	"gwcli/utilities/actionboiler"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	use     string   = "logout"
	short   string   = "Logout and end the session"
	long    string   = "Ends your current session and invalids your login token."
	aliases []string = []string{}
)

func NewLogoutAction() action.Pair {
	cmd, ba := actionboiler.NewBasicAction(use, short, long, aliases, func() (string, tea.Cmd) {
		connection.End()
		return "Successfully logged out", tea.Quit
	})
	return treeutils.GenerateAction(cmd, &ba)
}
