/**
 * gwcli is driven by mother/root.go's .Execute() method, which is called here
 */

package main

import (
	"gwcli/clilog"
	"gwcli/tree"

	"github.com/gravwell/gravwell/v3/ingest/log"
)

func init() {
	clilog.Init("gwcli.log", log.DEBUG) // TODO move this to root when flags are handled
}

func main() {
	tree.Execute()
}
