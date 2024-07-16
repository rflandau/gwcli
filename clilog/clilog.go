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

// recreate log.Level so other packages do not have to import it
type Level int

const (
	OFF      Level = 0
	DEBUG    Level = 1
	INFO     Level = 2
	WARN     Level = 3
	ERROR    Level = 4
	CRITICAL Level = 5
	FATAL    Level = 6
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

	Writer.Infof("Logger initialized at %v level, hostname %v", Writer.GetLevel(), Writer.Hostname())

	Writer.SetAppname(".")
	Writer.SetHostname(".") // autopopulates if empty

	return nil
}

// Writes the error to clilog.Writer and a secondary output, usually stderr
func Tee(lvl Level, alt io.Writer, str string) {
	alt.Write([]byte(str))
	switch lvl {
	case OFF:
	case DEBUG:
		Writer.Debug(str)
	case INFO:
		Writer.Info(str)
	case WARN:
		Writer.Warn(str)
	case ERROR:
		Writer.Error(str)
	case CRITICAL:
		Writer.Critical(str)
	case FATAL:
		Writer.Fatal(str)
	}
}

func Active(lvl Level) bool {
	return Writer.GetLevel() <= log.Level(lvl)
}
