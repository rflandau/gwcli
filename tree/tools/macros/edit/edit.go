package edit

import (
	"gwcli/action"
	"gwcli/clilog"
	"gwcli/connection"
	"gwcli/stylesheet"
	"gwcli/stylesheet/colorizer"
	"gwcli/utilities/treeutils"
	"gwcli/utilities/uniques"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

const listHeightMax = 40 // lines

// #region local styles
var (
	tiFieldRequiredSty = stylesheet.Header1Style
	tiFieldOptionalSty = stylesheet.Header2Style
)

// #endregion

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

	aflags := addtlFlags()
	cmd.Flags().AddFlagSet(&aflags)

	return treeutils.GenerateAction(cmd, newEditModel(fchFunc, addtlFlags))
}

func addtlFlags() pflag.FlagSet {
	fs := pflag.FlagSet{}
	fs.String("name", "", "new macro name")
	fs.String("description", "", "new macro description")
	fs.String("expansion", "", "new macro expansion")

	return fs
}

//#region interactive mode (model) implementation

type mode = uint8

const (
	quitting  mode = iota // mother should reassert
	selecting             // picking from a list of edit-able items
	editting              // item selected; currently altering
	idle                  // inactive
)

type titledTI struct {
	title    string          // field name to display next to the TI
	ti       textinput.Model // ti for user modifications
	required bool            // this field must not be empty
}

type editModel struct {
	mode          mode                 // current program state
	addtlFlagFunc func() pflag.FlagSet // function to generate flagset to parse field flags
	fs            pflag.FlagSet        // current state of the flagset
	width, height int

	fchFunc func() ([]types.SearchMacro, error) // func to retrieve each editable item
	data    []types.SearchMacro                 // data retrieved by fchFunc

	// selecting mode
	list list.Model // list displayed during `selecting` mode

	// editting mode
	ttis         []titledTI        // TIs will be displayed in array order
	tiIndex      int               // array index of active TI
	tiCount      int               // len(ttis)
	selectedData types.SearchMacro // item chosen from the list
	inputErr     string            // input is erroneous
	updateErr    string            // error occured performing the update
}

// Creates and returns a new edit model, ready for intreactive use.
//
// fchFunc must be a function that returns an array of editable structs.
//
// addtlFlagFunc may be nil or a function that returns a new flagset to parse/extract values from.
func newEditModel(fchFunc fetchFunc, addtlFlagFunc func() pflag.FlagSet) *editModel {
	em := &editModel{mode: idle, fchFunc: fchFunc, addtlFlagFunc: addtlFlagFunc}
	if em.addtlFlagFunc != nil {
		em.fs = em.addtlFlagFunc()
	} else {
		// set em.fs to the empty flagset
		clilog.Writer.Warnf("no flags were given to edit model")
		em.fs = pflag.FlagSet{}
	}

	return em
}

func (em *editModel) Update(msg tea.Msg) tea.Cmd {
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		em.width = msg.Width
		em.height = msg.Height
		em.list.SetSize(em.width, min(msg.Height-2, listHeightMax))
	}

	var cmd tea.Cmd
	// switch handling based on mode
	switch em.mode {
	case quitting:
		return nil
	case selecting:
		// switch on message type
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.Type == tea.KeySpace || msg.Type == tea.KeyEnter {
				em.selectedData = em.data[em.list.Index()]
				clilog.Writer.Debugf("editting macro %v", em.selectedData.Name)

				// transmute the selected item into a series of TIs
				em.ttis = transmuteStruct(em.selectedData, em.fs)
				em.tiCount = len(em.ttis)
				if em.tiCount < 1 {
					str := "no tis created by transmutation"
					clilog.Writer.Warnf(str)
					return nil
				}

				em.ttis[0].ti.Focus() // focus the first TI

				em.mode = editting
				return textinput.Blink
			}
		}
		em.list, cmd = em.list.Update(msg)
	case editting:
		// check for a submission via alt+enter
		if keymsg, ok := msg.(tea.KeyMsg); ok {
			switch keymsg.Type {
			case tea.KeyEnter:
				if keymsg.Alt {
					var populated bool = true
					for _, tti := range em.ttis { // check all required fields are populated
						if tti.required && strings.TrimSpace(tti.ti.Value()) == "" {
							em.inputErr = tti.title + " is required"
							populated = false
							break
						}
					}
					if populated {
						if invalMsg, err := upd(em.ttis, em.selectedData); err != nil {
							em.updateErr = err.Error()
						} else if invalMsg != "" {
							em.inputErr = invalMsg
						} else {
							// successfully updated; print a message and die
							em.mode = quitting
							return tea.Printf("Successfully updated %v %v",
								"macro", em.selectedData.Name)
						}
					}
					// if not populated, will fall through to normal update
				} else {
					em.nextTI()
				}
			case tea.KeyUp:
				em.previousTI()
			case tea.KeyDown:
				em.nextTI()
			}
		}

		// update tis
		cmds := make([]tea.Cmd, len(em.ttis))
		for i, tti := range em.ttis {
			em.ttis[i].ti, cmds[i] = tti.ti.Update(msg)
		}
		cmd = tea.Batch(cmds...)
	}

	return cmd
}

// Blur existing TI, select and focus previous (higher) TI
func (em *editModel) previousTI() {
	em.ttis[em.tiIndex].ti.Blur()
	em.tiIndex -= 1
	if em.tiIndex < 0 {
		em.tiIndex = em.tiCount - 1
	}
	em.ttis[em.tiIndex].ti.Focus()
}

// Blur existing TI, select and focus next (lower) TI
func (em *editModel) nextTI() {
	em.ttis[em.tiIndex].ti.Blur()
	em.tiIndex += 1
	if em.tiIndex >= em.tiCount {
		em.tiIndex = 0
	}
	em.ttis[em.tiIndex].ti.Focus()
}

// Takes the struct associated to the selected list item and transform its edit-able fields into a
// list of TIs.
func transmuteStruct(data types.SearchMacro, fs pflag.FlagSet) []titledTI {
	var tis []titledTI = make([]titledTI, 3)

	tis[0] = titledTI{ // name
		title:    "Name",
		ti:       stylesheet.NewTI(data.Name, false),
		required: true,
	}

	// check for name flag
	if x, err := fs.GetString("name"); err != nil {
		clilog.LogFlagFailedGet("name", err)
	} else if fs.Changed("name") {
		tis[0].ti.SetValue(x)
	}

	tis[1] = titledTI{ // description
		title:    "Description",
		ti:       stylesheet.NewTI(data.Description, false),
		required: true,
	}

	// check for description flag
	if x, err := fs.GetString("description"); err != nil {
		clilog.LogFlagFailedGet("description", err)
	} else if fs.Changed("description") {
		tis[1].ti.SetValue(x)
	}

	tis[2] = titledTI{ // description
		title:    "Expansion",
		ti:       stylesheet.NewTI(data.Expansion, false),
		required: true,
	}
	// check for description flag
	if x, err := fs.GetString("expansion"); err != nil {
		clilog.LogFlagFailedGet("expansion", err)
	} else if fs.Changed("expansion") {
		tis[2].ti.SetValue(x)
	}

	return tis
}

// Takes the populated TIs, validates their input, and updates the gravwell backend.
func upd(ttis []titledTI, data types.SearchMacro) (invalMsg string, err error) {
	// no need to nil check; all required fields are checked already

	// rebuild the struct for the update call
	for i, tti := range ttis {
		switch tti.title {
		case "Name":
			data.Name = strings.ToUpper(tti.ti.Value()) // name must always be uppercase
			ttis[i].ti.SetValue(data.Name)              // update it in case we return invalid or err
		case "Description":
			data.Description = tti.ti.Value()
		case "Expansion":
			data.Expansion = tti.ti.Value()
		}
	}

	// submit the updated struct
	return "", connection.Client.UpdateMacro(data)
}

func (em *editModel) View() string {
	var str string

	switch em.mode {
	case quitting:
		return ""
	case selecting:
		str = em.list.View() + "\n" +
			lipgloss.NewStyle().
				AlignHorizontal(lipgloss.Center).
				Width(em.width).
				Foreground(stylesheet.TertiaryColor).
				Render("Press enter to select")
	case editting:
		var sb strings.Builder
		for _, tti := range em.ttis {
			// color the title appropriately
			if tti.required {
				sb.WriteString(tiFieldRequiredSty.Render(tti.title + ": "))
			} else {
				sb.WriteString(tiFieldOptionalSty.Render(tti.title + ": "))
			}
			sb.WriteString(tti.ti.View() + "\n")
		}
		sb.WriteString(colorizer.SubmitString("alt+enter", em.inputErr, "", em.width))
		str = sb.String()
	}
	return str
}

func (em *editModel) Done() bool {
	return em.mode == quitting
}

func (em *editModel) Reset() error {
	em.mode = idle
	em.data = nil
	if em.addtlFlagFunc != nil {
		em.fs = em.addtlFlagFunc()
	} else {
		em.fs = pflag.FlagSet{}
	}

	// selecting mode
	em.list = list.Model{}

	// editting mode
	em.ttis = nil
	em.tiIndex = 0
	em.tiCount = 0
	em.selectedData = types.SearchMacro{}
	em.inputErr = ""
	em.updateErr = ""

	return nil
}

func (em *editModel) SetArgs(_ *pflag.FlagSet, tokens []string) (
	invalid string, onStart tea.Cmd, err error,
) {
	// parse the flags, save them for later, when TIs are created
	if err := em.fs.Parse(tokens); err != nil {
		return "", nil, err
	}

	// fetch edit-able macros
	if em.data, err = em.fchFunc(); err != nil {
		return
	}

	var dataCount = len(em.data)

	// check for a lack of data
	if dataCount < 1 { // die
		em.mode = quitting
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
	em.list = list.New(itms, list.NewDefaultDelegate(), 80, listHeightMax)

	em.mode = selecting

	return "", uniques.FetchWindowSize, nil
}

//#endregion interactive mode (model) implementation

type macroItem struct {
	title, description string
}

var _ list.DefaultItem = macroItem{}

func (mi macroItem) FilterValue() string { return mi.title }
func (mi macroItem) Title() string       { return mi.title }
func (mi macroItem) Description() string { return mi.description }
