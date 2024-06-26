package myinfo

import (
	"fmt"
	"gwcli/action"
	"gwcli/clilog"
	"gwcli/connection"
	"gwcli/stylesheet"
	"gwcli/utilities/scaffold"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	use     string   = "myinfo"
	short   string   = "Information about the current user and session"
	long    string   = "Displays your accounts information and capabilities"
	aliases []string = []string{}
)

func NewMyInfoAction() action.Pair {
	return scaffold.NewBasicAction(use, short, long, aliases, func() (string, tea.Cmd) {
		ud, err := connection.Client.MyInfo()
		if err != nil {
			s := fmt.Sprintf("Unable to determine user info: %v", err)
			clilog.Writer.Error(s)
			return s, nil
		}

		sty := stylesheet.Header1Style.Bold(false)
		out := fmt.Sprintf("%v, %v, %v\n%s: %v\n%s: %v\n%s: %v", ud.Name, ud.User, ud.Email,
			sty.Render("Groups"), ud.Groups,
			sty.Render("Capabilities"), ud.CapabilityList(),
			sty.Render("Admin"), ud.Admin)

		return out, nil
	})
}
