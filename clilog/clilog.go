package clilog

import (
	"io"

	"github.com/gravwell/gravwell/v3/ingest/log"
)

var Writer *log.Logger

/**
 * Initializes Writer, the logging singleton.
 * Safe (ineffectual) if the writer has already been initialized.
 */
func Init(path string, lvl string) error {
	// TODO make the logger terse by default
	var err error
	if Writer != nil {
		return nil
	}

	level, err := log.LevelFromString(lvl)
	if err != nil {
		return err
	}
	Writer, err = log.NewFile(path)
	if err != nil {
		Writer.Close()
		return err
	}

	if err = Writer.SetLevel(level); err != nil {
		Writer.Close()
		return err
	}

	return nil
}

// Writes the error to clilog.Writer and a secondary output, usually stderr
func TeeError(alt io.Writer, str string) {
	Writer.Debugf(str)
	alt.Write([]byte(str))
}

