// A simple action to tell the user whether or not they are logged in as an admin.
package admin

import (
	"fmt"
	"gwcli/action"
	"gwcli/connection"
	"gwcli/utilities/scaffold"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/pflag"
)

var (
	use     string   = "admin"
	short   string   = "Prints your admin status"
	long    string   = "Displays whether or not your current user has admin permissions"
	aliases []string = []string{}
)

func NewAdminAction() action.Pair {
	return scaffold.NewBasicAction(use, short, long, aliases, func(*pflag.FlagSet) (string, tea.Cmd) {
		var not string
		if !connection.Client.AdminMode() {
			not = " not"
		}
		return fmt.Sprintf("You are%v in admin mode", not), nil
	}, nil)
}
