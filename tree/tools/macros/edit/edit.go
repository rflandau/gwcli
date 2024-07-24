package edit

import (
	"gwcli/action"
	"gwcli/connection"
	"gwcli/treeutils"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gravwell/gravwell/v3/client/types"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// user must select a macro to edit
// -> list all macros or provide ID by flag
// display one TI per edittable field
// -> prepopulate current information in each field
// -> may need a mapperFunc so a user can transmute values from fecth func into TI values
// user makes whatever edits are necessary and submits
// -> basically identically to a pre-populated create
// transmute data in TIs back into the original struct
// update

// combination of delete's listing/selection capabilities and create's TI interface

type fetchFunc = func() ([]types.SearchMacro, error)

func NewMacroEditAction() action.Pair {
	cmd := treeutils.NewActionCommand("edit", "edit a macro", "edit/alter an existing macro",
		[]string{"e"}, func(c *cobra.Command, s []string) {})

	// need one flag per edittable field
	// TODO

	// fetch available macros in SetArgs or run, in case macros are updated prior to usage
	// TODO run fetch in run() if an explicit ID is not given

	fchFunc := func() ([]types.SearchMacro, error) {
		return connection.Client.GetUserMacros(connection.MyInfo.UID)
	}

	return treeutils.GenerateAction(cmd, newEditModel(fchFunc))
}

//#region interactive mode (model) implementation

type mode = uint8

const (
	quitting  mode = iota // mother should reassert
	selecting             // picking from a list of edit-able items
	editting              // item selected; currently altering
	idle                  // inactive
)

type editModel struct {
	mode mode // current program state

	fchFunc func() ([]types.SearchMacro, error) // func to retrieve each editable item
	data    []types.SearchMacro                 // data retrieved by fchFunc
	list    list.Model                          // list displayed during `selecting` mode

}

func newEditModel(fchFunc fetchFunc) *editModel {
	return &editModel{fchFunc: fchFunc}
}

func (em *editModel) Update(msg tea.Msg) tea.Cmd {
	// switch handling based on mode
	switch em.mode {
	case quitting:
		return nil
	case selecting:
		// switch on message type
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			// TODO if the user is able to return to selection, this must update no matter the mode
			em.list.SetSize(msg.Width, msg.Height)
		}
		var cmd tea.Cmd
		em.list, cmd = em.list.Update(msg)
		return cmd
	}

	return nil
}

func (em *editModel) View() string {
	return ""
}

func (em *editModel) Done() bool {
	return em.mode == quitting
}

func (em *editModel) Reset() error {
	em.mode = idle
	em.data = nil
	em.list = list.Model{}

	return nil
}

func (em *editModel) SetArgs(*pflag.FlagSet, []string) (invalid string, onStart tea.Cmd, err error) {
	// fetch edit-able macros
	if em.data, err = em.fchFunc(); err != nil {
		return
	}

	var dataCount = len(em.data)

	// check for a lack of data
	if dataCount < 1 { // die
		return "", tea.Println("You have no macros that can be editted"), nil
	}

	// transmute data into list items
	var itms []list.Item = make([]list.Item, dataCount)
	for i, m := range em.data {
		itms[i] = macroItem{
			title:       m.Name,
			description: m.Description,
		}
	}

	// generatelist
	em.list = list.New(itms, list.NewDefaultDelegate(), 80, 40)

	return "", nil, nil
}

//#endregion interactive mode (model) implementation

type macroItem struct {
	title, description string
}

var _ list.DefaultItem = macroItem{}

func (mi macroItem) FilterValue() string { return mi.title }
func (mi macroItem) Title() string       { return mi.title }
func (mi macroItem) Description() string { return mi.description }
