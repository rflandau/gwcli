package scaffold

import "fmt"

type outputFormat uint

const (
	json outputFormat = iota
	csv
	table
	unknown
)

func (f outputFormat) String() string {
	switch f {
	case json:
		return "JSON"
	case csv:
		return "CSV"
	case table:
		return "table"
	}
	return fmt.Sprintf("unknown format (%d)", f)
}
