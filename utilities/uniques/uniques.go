// uniques contains global constants that must be referenced across multiple packages
package uniques

import (
	"fmt"
	"gwcli/clilog"
)

const (
	// the string format the Gravwell client requires
	SearchTimeFormat = "2006-01-02T15:04:05.999999999Z07:00"
)

// Returns a *newline-suffixed* string about ignoring a flag due to causeFlag's existance (or the
// empty string if WARN would not be printed).
func WarnFlagIgnore(ignoredFlag, causeFlag string) string {
	if clilog.Active(clilog.WARN) {
		return fmt.Sprintf("WARN: ignoring flag --%v due to --%v\n", ignoredFlag, causeFlag)
	}
	return ""
}
