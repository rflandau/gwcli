package stylesheet

import (
	"gwcli/action"

	"github.com/spf13/cobra"
)

func ColorCommandName(c *cobra.Command) string {
	if action.Is(c) {
		return ActionStyle.Render(c.Name())
	} else {
		return NavStyle.Render(c.Name())
	}
}
