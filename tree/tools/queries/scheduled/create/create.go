package create

import (
	"gwcli/action"
	"gwcli/connection"
	"gwcli/stylesheet"
	ft "gwcli/stylesheet/flagtext"
	"gwcli/utilities/scaffold/scaffoldcreate"
	"gwcli/utilities/uniques"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/spf13/pflag"
)

const ( // field keys
	kname = "name"
	kdesc = "desc"
	kfreq = "freq"
	kqry  = "qry"
	kdur  = "dur"
)

func NewQueriesScheduledCreateAction() action.Pair {
	fields := scaffoldcreate.Config{
		kname: scaffoldcreate.NewField(true, "name", 100),
		kdesc: scaffoldcreate.NewField(false, "description", 90),
		kdur:  scaffoldcreate.NewField(true, "duration", 140),
		kqry:  scaffoldcreate.NewField(true, "query", 150),
		kfreq: scaffoldcreate.Field{ // manually build so we have more control
			Required:      true,
			Title:         "frequency",
			Usage:         ft.Usage.Frequency,
			Type:          scaffoldcreate.Text,
			FlagName:      ft.Name.Frequency, // custom flag name
			FlagShorthand: 'f',
			DefaultValue:  "", // no default value
			Order:         50,
			CustomTIFuncInit: func() textinput.Model {
				ti := stylesheet.NewTI("", false)
				ti.Placeholder = "* * * * *"
				ti.Validate = uniques.CronRuneValidator
				return ti
			},
		},
	}

	return scaffoldcreate.NewCreateAction("scheduled query", fields, create, nil)
}

func create(_ scaffoldcreate.Config, vals map[string]string, _ *pflag.FlagSet) (any, string, error) {
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
