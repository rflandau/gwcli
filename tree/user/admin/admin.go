package admin

import (
	"fmt"
	"gwcli/action"
	"gwcli/connection"
	"gwcli/treeutils"
	"gwcli/utilities/actionboiler"
)

var (
	use     string   = "admin"
	short   string   = "Prints your admin status"
	long    string   = "Displays whether or not your current user has admin permissions"
	aliases []string = []string{}
)

func NewAdminAction() action.Pair {
	cmd, ba := actionboiler.NewBasicCmd(use, short, long, aliases, func() string {
		var not string
		if !connection.Client.AdminMode() {
			not = " not"
		}
		return fmt.Sprintf("You are%v an admin", not)
	})
	return treeutils.GenerateAction(cmd, &ba)
}
