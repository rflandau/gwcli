// A simple logout action that logs out the current user and ends the program
package logout

import (
	"gwcli/action"
	"gwcli/connection"
	"gwcli/treeutils"
	"gwcli/utilities/actionboiler"
)

var (
	use     string   = "logout"
	short   string   = "Logout and end the session"
	long    string   = "Ends your current session and invalids your login token."
	aliases []string = []string{}
)

func NewLogoutAction() action.Pair {
	cmd, ba := actionboiler.NewBasicCmd(use, short, long, aliases, func() string {
		connection.End()
		// TODO pass back a tea.Quit
		return "Successfully logged out"
	})
	return treeutils.GenerateAction(cmd, &ba)
}
