package create

import (
	"fmt"
	"gwcli/action"
	"gwcli/connection"
	"gwcli/utilities/scaffold/scaffoldcreate"
	"strings"

	"github.com/gravwell/gravwell/v3/client/types"
)

const (
	kname   = "name"
	kdesc   = "desc"
	kmodule = "module"
	ktags   = "tags"
	kparams = "params"
	kargs   = "args"
	klabels = "labels"
)

func NewExtractorsCreateAction() action.Pair {
	fields := scaffoldcreate.Config{
		kname: scaffoldcreate.Field{
			Required:      true,
			Title:         "name",
			Usage:         "name of the new extractor",
			Type:          scaffoldcreate.Text,
			FlagName:      "name",
			FlagShorthand: 'n',
			TI: struct {
				Order       int
				Placeholder string
				Validator   func(s string) error
			}{
				Order: 100,
			},
		},
		kdesc: scaffoldcreate.Field{
			Required:      true,
			Title:         "description",
			Usage:         "description of the new extractor",
			Type:          scaffoldcreate.Text,
			FlagName:      "desc",
			FlagShorthand: 'd',
			TI: struct {
				Order       int
				Placeholder string
				Validator   func(s string) error
			}{
				Order: 90,
			},
		},
		kmodule: scaffoldcreate.Field{
			Required:      true,
			Title:         "module",
			Usage:         "",
			Type:          scaffoldcreate.Text,
			FlagName:      "module",
			FlagShorthand: 'm',
			DefaultValue:  "",
			TI: struct {
				Order       int
				Placeholder string
				Validator   func(s string) error
			}{
				Order: 80,
			},
		},
		ktags: scaffoldcreate.Field{
			Required:      true,
			Title:         "tags",
			Usage:         "tags this ax will extract from. There can only be one extractor per tag.",
			Type:          scaffoldcreate.Text,
			FlagName:      "tags",
			FlagShorthand: 't',
			DefaultValue:  "",
			TI: struct {
				Order       int
				Placeholder string
				Validator   func(s string) error
			}{
				Order:       70,
				Placeholder: "tag1,tag2,tag3",
			},
		},
		kparams: scaffoldcreate.Field{
			Required:     false,
			Title:        "params/regex",
			Usage:        "",
			Type:         scaffoldcreate.Text,
			FlagName:     "params",
			DefaultValue: "",
			TI: struct {
				Order       int
				Placeholder string
				Validator   func(s string) error
			}{
				Order: 60,
				//Placeholder: "",
			},
		},
		kargs: scaffoldcreate.Field{
			Required:     false,
			Title:        "arguments/options",
			Usage:        "arguments/options on this ax",
			Type:         scaffoldcreate.Text,
			FlagName:     "args",
			DefaultValue: "",
			TI: struct {
				Order       int
				Placeholder string
				Validator   func(s string) error
			}{
				Order: 50,
				//Placeholder: "",
			},
		},
		klabels: scaffoldcreate.Field{
			Required:     false,
			Title:        "labels/categories",
			Usage:        "arguments/options on this ax",
			Type:         scaffoldcreate.Text,
			FlagName:     "labels",
			DefaultValue: "",
			TI: struct {
				Order       int
				Placeholder string
				Validator   func(s string) error
			}{
				Order:       40,
				Placeholder: "label1,label2,label3",
			},
		},
	}

	return scaffoldcreate.NewCreateAction("extractor", fields, create)
}

func create(_ scaffoldcreate.Config, vals scaffoldcreate.Values) (any, string, error) {
	// no need to nil check; Required boolean enforces that for us
	axd := types.AXDefinition{
		Name:   vals[kname],
		Desc:   vals[kdesc],
		Module: vals[kmodule],
		Tags:   strings.Split(strings.Replace(vals[ktags], " ", "", -1), ","),
		Params: vals[kparams],
		Args:   vals[kargs],
		Labels: strings.Split(strings.Replace(vals[klabels], " ", "", -1), ","),
	}

	id, wrs, err := connection.Client.AddExtraction(axd)

	if len(wrs) > 0 {
		var invSB strings.Builder
		for _, wr := range wrs {
			invSB.WriteString(fmt.Sprintf("%v: %v\n", wr.Name, wr.Err))
		}
		return 0, invSB.String(), nil
	}

	return id, "", err
}
