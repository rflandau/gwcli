package systems

import (
	"gwcli/action"
	"gwcli/tree/systems/systemsactions"
	"gwcli/treeutils"

	"github.com/spf13/cobra"
)

var (
	use     string   = "systems"
	short   string   = "Systems & Health submenu"
	long    string   = "Actions associated to monitoring the health and status of the system rit large"
	aliases []string = []string{"system", "health"}
)

func NewSystemsNav() *cobra.Command {
	return treeutils.GenerateNav(use, short, long, aliases, nil, []action.Pair{systemsactions.NewHardwareList(), systemsactions.NewDiskInfo()})
}
