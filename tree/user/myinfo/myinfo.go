package myinfo

import (
	"fmt"
	"gwcli/action"
	"gwcli/clilog"
	"gwcli/connection"
	"gwcli/stylesheet"
	"gwcli/utilities/scaffold"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gravwell/gravwell/v3/client/types"
	"github.com/gravwell/gravwell/v3/utils/weave"
	"github.com/spf13/pflag"
)

var (
	use     string   = "myinfo"
	short   string   = "Information about the current user and session"
	long    string   = "Displays your accounts information and capabilities"
	aliases []string = []string{}
)

func NewUserMyInfoAction() action.Pair {

	return scaffold.NewBasicAction(use, short, long, aliases, func(fs *pflag.FlagSet) (string, tea.Cmd) {
		ud, err := connection.Client.MyInfo()
		if err != nil {
			s := fmt.Sprintf("Unable to determine user info: %v", err)
			clilog.Writer.Error(s)
			return s, nil
		}

		if asCSV, err := fs.GetBool("csv"); err != nil {
			s := fmt.Sprintf("Failed to fetch csv flag: %v", err)
			clilog.Writer.Error(s)
			return s, nil
		} else if asCSV {
			return weave.ToCSV([]types.UserDetails{ud}, []string{
				"UID",
				"User",
				"Name",
				"Email",
				"Admin",
				"Locked",
				"TS",
				"DefaultGID",
				"Groups",
				"Hash",
				"Synced",
				"CBAC"}), nil
		}

		sty := stylesheet.Header1Style.Bold(false)
		out := fmt.Sprintf("%v, %v, %v\n%s: %v\n%s: %v\n%s: %v", ud.Name, ud.User, ud.Email,
			sty.Render("Groups"), ud.Groups,
			sty.Render("Capabilities"), ud.CapabilityList(),
			sty.Render("Admin"), ud.Admin)

		return out, nil
	}, flags)
}

func flags() pflag.FlagSet {
	fs := pflag.FlagSet{}
	fs.Bool("csv", false, "display as CSV")
	return fs
}
