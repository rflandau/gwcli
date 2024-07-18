package create

import (
	"gwcli/action"
	"gwcli/connection"
	"gwcli/utilities/scaffold/scaffoldcreate"
	"gwcli/utilities/uniques"
	"time"
)

const ( // field keys
	kname = "name"
	kdesc = "desc"
	kfreq = "freq"
	kqry  = "qry"
	kdur  = "dur"
)

var (
	aliases []string = []string{}
)

func NewQueriesScheduledCreateAction() action.Pair {
	fields := scaffoldcreate.FieldMap{
		kname: scaffoldcreate.NewField(true, "name"),
		kdesc: scaffoldcreate.NewField(false, "description"),
		kfreq: scaffoldcreate.NewField(true, "frequency"),
		kqry:  scaffoldcreate.NewField(true, "query"),
	}

	// assign validator functions
	durField := scaffoldcreate.NewField(true, "duration")
	durField.TI.Validator = uniques.CronRuneValidator
	durField.TI.Placeholder = "* * * * *"
	fields[kdur] = durField

	return scaffoldcreate.NewCreateAction(aliases,
		"scheduled query",
		fields, create)
}

func create(fields scaffoldcreate.FieldMap) (any, string, error) {
	name := fields[kname].Value
	desc := fields[kdesc].Value
	freq := fields[kfreq].Value
	qry := fields[kqry].Value
	durString := fields[kdur].Value

	dur, err := time.ParseDuration(durString)
	if err != nil { // report as invalid parameter, not an error
		return nil, err.Error(), nil
	}

	return connection.CreateScheduledSearch(name, desc, freq, qry, dur)
}
