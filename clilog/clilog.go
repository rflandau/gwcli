/**
 * clilog provides the logger for gwcli in the form of a logging singleton:
 * Writer.
 *
 * It is basically a singleton wrapper of the gravwell ingest logger.
 * While the underlying ingest logger appears to be thread-safe, clilog's helper
 * functions are not necessarily.
 */
package clilog

import (
	"io"

	"github.com/gravwell/gravwell/v3/ingest/log"
)

var Writer *log.Logger

// Initializes Writer, the logging singleton.
// Safe (ineffectual) if the writer has already been initialized.
func Init(path string, lvl string) error {
	var err error
	if Writer != nil {
		return nil
	}

	Writer, err = log.NewFile(path)
	if err != nil {
		Writer.Close()
		return err
	}

	if err = Writer.SetLevelString(lvl); err != nil {
		Writer.Close()
		return err
	}

	Writer.Debugf("Logger initialized at %v level, hostname %v", Writer.GetLevel(), Writer.Hostname())

	Writer.SetAppname(".")
	Writer.SetHostname(".") // autopopulates if empty

	return nil
}

// Writes the error to clilog.Writer and a secondary output, usually stderr
func TeeError(alt io.Writer, str string) {
	Writer.Debugf(str)
	alt.Write([]byte(str))
}
