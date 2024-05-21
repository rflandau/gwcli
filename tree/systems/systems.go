package systems

import (
	"gwcli/tree/systems/actions"
	"gwcli/treeutils"

	"github.com/spf13/cobra"
)

var (
	use     string   = "systems"
	short   string   = "Systems & Health submenu"
	long    string   = "Actions associated to monitoring the health and status of the system rit large"
	aliases []string = []string{"system", "health"}
)

func GenerateTree() *cobra.Command {
	return treeutils.GenerateNav(use, short, long, aliases, actions.NewHardwareCmd())
}

/*
func(cmd *cobra.Command, args []string) {
			fmt.Printf("PreRun cmd: '%#v'\n", cmd)
			noInteractive, err := cmd.Flags().GetBool("no-interactive")
			fmt.Println("noInteractive mode is", noInteractive)
			if err != nil {
				panic(err)
			}
			if noInteractive == false {
				fmt.Println("noInteractive mode is", noInteractive)
				cmd.Run = func(cmd *cobra.Command, args []string) {
					fmt.Println("systems called") // TODO
				}
			}
		}
*/
