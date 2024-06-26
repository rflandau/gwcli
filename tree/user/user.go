package user

import (
	"gwcli/action"
	"gwcli/tree/user/admin"
	"gwcli/tree/user/logout"
	"gwcli/tree/user/myinfo"
	"gwcli/treeutils"

	"github.com/spf13/cobra"
)

var (
	use     string   = "user"
	short   string   = "Manage your user and profile"
	long    string   = "View and edit properties of your current, logged in user."
	aliases []string = []string{"self"}
)

func GenerateTree() *cobra.Command {
	return treeutils.GenerateNav(use, short, long, aliases, nil,
		[]action.Pair{logout.NewLogoutAction(), admin.NewAdminAction(), myinfo.NewMyInfoAction()})
}
