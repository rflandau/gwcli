package delete

import (
	"errors"
	"gwcli/action"
	"gwcli/clilog"
	"gwcli/connection"
	"gwcli/utilities/scaffold/scaffolddelete"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/gravwell/gravwell/v3/client/types"
)

func NewExtractorDeleteAction() action.Pair {
	return scaffolddelete.NewDeleteAction([]string{}, "extractor", "extractors",
		del, fetch)
}

func del(dryrun bool, id uuid.UUID) error {
	if dryrun {
		_, err := connection.Client.GetExtraction(id.String())
		return err
	}
	if wrs, err := connection.Client.DeleteExtraction(id.String()); err != nil {
		return err
	} else if wrs != nil {
		var sb strings.Builder
		sb.WriteString("failed to delete ax with warning(s):")
		for _, wr := range wrs {
			sb.WriteString("\n" + wr.Err.Error())
		}
		clilog.Writer.Warn(sb.String())
		return errors.New(sb.String())
	}
	return nil
}

func fetch() ([]scaffolddelete.Item[uuid.UUID], error) {
	axs, err := connection.Client.GetExtractions()
	if err != nil {
		return nil, err
	}
	slices.SortFunc(axs, func(a1, a2 types.AXDefinition) int {
		return strings.Compare(a1.Name, a2.Name)
	})
	var items = make([]scaffolddelete.Item[uuid.UUID], len(axs))
	for i := range axs {
		items[i] = axItem{id: axs[i].UUID, name: axs[i].Name, desc: axs[i].Desc}
	}

	return items, nil
}

type axItem struct {
	id   uuid.UUID
	name string
	desc string
}

var _ scaffolddelete.Item[uuid.UUID] = axItem{}

func (ai axItem) ID() uuid.UUID       { return ai.id }
func (ai axItem) FilterValue() string { return ai.name }
func (ai axItem) String() string {
	return ai.name + ": " + ai.id.String() + "\n" +
		ai.desc
}
