// A simple action to tell the user whether or not they are logged in as an admin.
package admin

import (
	"fmt"
	"gwcli/action"
	"gwcli/connection"
	"gwcli/utilities/actionboiler"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	use     string   = "admin"
	short   string   = "Prints your admin status"
	long    string   = "Displays whether or not your current user has admin permissions"
	aliases []string = []string{}
)

func NewAdminAction() action.Pair {
	return actionboiler.NewBasicAction(use, short, long, aliases, func() (string, tea.Cmd) {
		var not string
		if !connection.Client.AdminMode() {
			not = " not"
		}
		return fmt.Sprintf("You are%v an admin", not), nil
	})
}
