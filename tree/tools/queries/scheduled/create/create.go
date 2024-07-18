package create

import (
	"gwcli/action"
	"gwcli/connection"
	"gwcli/stylesheet"
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
	fields := scaffoldcreate.Config{
		kname: scaffoldcreate.NewField(true, "name"),
		kdesc: scaffoldcreate.NewField(false, "description"),
		kfreq: scaffoldcreate.NewField(true, "frequency"),
		kqry:  scaffoldcreate.NewField(true, "query"),
		kdur: scaffoldcreate.Field{
			Required:     true,
			Title:        "duration",
			Usage:        stylesheet.FlagDurationDesc,
			Type:         scaffoldcreate.Text,
			FlagName:     scaffoldcreate.DeriveFlagName("duration"),
			DefaultValue: "", // no default value
			TI: struct {
				Placeholder string
				Validator   func(s string) error
			}{Placeholder: "* * * * *", Validator: uniques.CronRuneValidator},
		},
	}

	// assign validator functions
	//durField := scaffoldcreate.NewField(true, "duration")

	return scaffoldcreate.NewCreateAction(aliases,
		"scheduled query",
		fields, create)
}

func create(_ scaffoldcreate.Config, vals map[string]string) (any, string, error) {
	var (
		name      = vals[kname]
		desc      = vals[kdesc]
		freq      = vals[kfreq]
		qry       = vals[kqry]
		durString = vals[kdur]
	)
	dur, err := time.ParseDuration(durString)
	if err != nil { // report as invalid parameter, not an error
		return nil, err.Error(), nil
	}

	return connection.CreateScheduledSearch(name, desc, freq, qry, dur)
}
