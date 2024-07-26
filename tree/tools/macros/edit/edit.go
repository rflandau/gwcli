package edit

import (
	"errors"
	"fmt"
	"gwcli/action"
	"gwcli/clilog"
	"gwcli/connection"
	"gwcli/stylesheet"
	"gwcli/stylesheet/colorizer"
	"gwcli/utilities/treeutils"
	"gwcli/utilities/uniques"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gravwell/gravwell/v3/client/types"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const listHeightMax = 40 // lines

//#region function signatures

// function to get the specific, edit-able struct and skip list/selecting mode.
type selectFunction = func(id uint64) (
	item types.SearchMacro, err error,
)

// function to fetch all edit-able structs
type fetchAllFunction = func() (
	[]types.SearchMacro, error,
)

// Function to retrieve the struct value associated to the field key without reflection.
// This is probably a switch statement that maps (key -> item.X).
//
// Sister to setFieldFunction.
type getFieldFunction = func(item types.SearchMacro, fieldKey string) (
	string, error,
)

// Function to set the struct value associated to the field key without reflection.
// This is probably a switch statement that maps (key -> item.X).
//
// Sister to getFieldFunction.
type setFieldFunction = func(item *types.SearchMacro, fieldKey, val string) error

// function to perform the actual update of the data on the GW instance
type updateStructFunction = func(data *types.SearchMacro) (
	title, invalidMsg string, err error,
)

// Set of all functions, to make it easier to pass them around internally.
// All fields are required.
type functionSet struct {
	selectFunc selectFunction
	fchFunc    fetchAllFunction
	getFunc    getFieldFunction
	setFunc    setFieldFunction
	updFunc    updateStructFunction
}

//#endregion

type Config = map[string]Field

// #region local styles
var (
	tiFieldRequiredSty = stylesheet.Header1Style
	tiFieldOptionalSty = stylesheet.Header2Style
)

// #endregion

func NewMacroEditAction() action.Pair {
	// TODO passed as parameter
	funcs := functionSet{
		selectFunc: getMacro,
		fchFunc: func() ([]types.SearchMacro, error) {
			return connection.Client.GetUserMacros(connection.MyInfo.UID)
		},
		getFunc: func(item types.SearchMacro, fieldKey string) (string, error) {
			switch fieldKey {
			case "name":
				return item.Name, nil
			case "description":
				return item.Description, nil
			case "expansion":
				return item.Expansion, nil
			}

			return "", fmt.Errorf("unknown field key: %v", fieldKey)
		},
		setFunc: func(item *types.SearchMacro, fieldKey, val string) error {
			switch fieldKey {
			case "name":
				item.Name = val
			case "description":
				item.Description = val
			case "expansion":
				item.Expansion = val
			default:
				return fmt.Errorf("unknown field key: %v", fieldKey)
			}
			return nil
		},
	}
	// check that all functions are given

	cmd := treeutils.NewActionCommand("edit", "edit a macro", "edit/alter an existing macro",
		[]string{"e"}, func(c *cobra.Command, s []string) {})

	// TODO temporary
	cfg := Config{
		"name": Field{
			Required: true,
			Title:    "Name",
			Usage:    stylesheet.FlagDescMacroName,
			FlagName: uniques.DeriveFlagName("name"),
			Order:    100,
		},
		"description": Field{
			Required: true,
			Title:    "Description",
			Usage:    stylesheet.FlagDescMacroDesc,
			FlagName: uniques.DeriveFlagName("description"),
			Order:    80,
		},
		"expansion": Field{
			Required: true,
			Title:    "Expansion",
			Usage:    stylesheet.FlagDescMacroExpansion,
			FlagName: uniques.DeriveFlagName("expansion"),
			Order:    60,
		},
	}

	cmd.Run = func(cmd *cobra.Command, args []string) {
		var err error
		// hard branch on script mode
		var script bool
		if script, err = cmd.Flags().GetBool("script"); err != nil {
			clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error()+"\n")
			return
		}
		if script {
			runNonInteractive(cmd, args, getMacro, macroTranslation, upd, cfg)
		}
	}

	// assign base flags
	flags, aflags := flags(), addtlFlags()
	cmd.Flags().AddFlagSet(&flags)
	cmd.Flags().AddFlagSet(&aflags)

	return treeutils.GenerateAction(cmd, newEditModel(
		cfg,
		fchFunc,
		getMacro,
		macroTranslation,
		addtlFlags))
}

const ( // local flag names
	flagID = "id"
)

// run helper function
// runNonInteractive is the --script portion of edit's runFunc.
// It requires --id be set and is ineffectual if an addtl/field flag was no given.
func runNonInteractive(cmd *cobra.Command, args []string,
	selectFunc selectFunction, getFFunc getFieldFunction,
	updFunc updateStructFunction, cfg Config) {
	var err error
	var (
		id   uint64
		zero uint64
		itm  types.SearchMacro
	)
	if id, err = cmd.Flags().GetUint64(flagID); err != nil {
		clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error()+"\n")
		return
	}
	if id == zero { // id was not given
		fmt.Fprintln(cmd.OutOrStdout(), "--id is required in script mode")
		return
	}
	// check for other flags; no point in updating if there is nothing to change
	// TODO

	// get the macro to edit
	if itm, err = selectFunc(id); err != nil {
		clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error()+"\n")
		return
	}

	var fieldUpdated bool   // was a value actually changed?
	for k, v := range cfg { // edit each field using their flag value
		// get current value
		curVal, err := getFFunc(itm, k)
		if err != nil {
			clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error()+"\n")
			return
		}
		var newVal string = curVal
		if cmd.Flags().Changed(v.FlagName) { // flag presumably updates the field
			if x, err := cmd.Flags().GetString(v.FlagName); err != nil {
				clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error()+"\n")
				return
			} else {
				newVal = x
			}
		}

		if newVal != curVal { // update the struct
			fieldUpdated = true // note if a change occured

		}

	}

	if !fieldUpdated { // only bother to update if at least one field was changed
		clilog.Tee(clilog.INFO, cmd.OutOrStdout(), "no field would be updated; quitting...")
		return
	}

	// perform the actual update
	updFunc()

}

// base flagset always available to edit actions
func flags() pflag.FlagSet {
	fs := pflag.FlagSet{}
	fs.Uint64("id", 0, "id of the macro to edit")
	return fs
}

func addtlFlags() pflag.FlagSet {
	fs := pflag.FlagSet{}
	fs.String(stylesheet.FlagNameMacroName, "", stylesheet.FlagDescMacroName)
	fs.String(stylesheet.FlagNameMacroDesc, "", stylesheet.FlagDescMacroDesc)
	fs.String(stylesheet.FlagNameMacroExpansion, "", stylesheet.FlagDescMacroExpansion)

	return fs
}

func getMacro(id uint64) (types.SearchMacro, error) {
	return connection.Client.GetMacro(id)
}

//#region interactive mode (model) implementation

type mode = uint8

const (
	quitting  mode = iota // mother should reassert
	selecting             // picking from a list of edit-able items
	editting              // item selected; currently altering
	idle                  // inactive
)

type keyedTI struct {
	key string          // key to look up the related field in the Config
	ti  textinput.Model // ti for user modifications
}

type editModel struct {
	mode          mode                 // current program state
	addtlFlagFunc func() pflag.FlagSet // function to generate flagset to parse field flags
	fs            pflag.FlagSet        // current state of the flagset
	width, height int

	cfg Config // RO configuration provided by the caller

	data      []types.SearchMacro // data retrieved by fchFunc
	getFunc   selectFunction      // func to retrieve a specified editable item
	transFunc getFieldFunction    // func to retrieve field values from a struct

	// selecting mode
	fchFunc fetchAllFunction // func to retrieve each editable item
	list    list.Model       // list displayed during `selecting` mode

	// editting mode
	orderedKTIs  []keyedTI         // TIs will be displayed in array order
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
// getFunc must be the function for getting a single struct, given id.
//
// addtlFlagFunc may be nil or a function that returns a new flagset to parse/extract values from.
func newEditModel(cfg Config,
	fchFunc fetchAllFunction,
	selectFunc selectFunction,
	getFFunc getFieldFunction,
	updFunc updateStructFunction,
	addtlFlagFunc func() pflag.FlagSet) *editModel {
	// sanity check required arguments
	if cfg == nil {
		panic("Configuration cannot be nil")
	}
	if fchFunc == nil {
		panic("fetch function cannot be nil")
	}

	if selectFunc == nil {
		panic("get function cannot be nil")
	}

	if getFFunc == nil {
		panic("translation function cannot be nil")
	}

	em := &editModel{mode: idle,
		cfg:           cfg,
		fchFunc:       fchFunc,
		addtlFlagFunc: addtlFlagFunc,
		getFunc:       selectFunc,
		transFunc:     getFFunc}
	em.fs = flags()
	if em.addtlFlagFunc != nil {
		aflags := em.addtlFlagFunc()
		em.fs.AddFlagSet(&aflags)
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
				if err := em.enterEditMode(); err != nil {
					em.mode = quitting
					clilog.Writer.Errorf("%v", err)
					return tea.Println(err.Error())
				}
				return textinput.Blink
			}
		}
		em.list, cmd = em.list.Update(msg)
	case editting:
		// check for a submission via alt+enter
		if keymsg, ok := msg.(tea.KeyMsg); ok {
			em.inputErr = "" // clear input errors on new key input
			switch keymsg.Type {
			case tea.KeyEnter:
				if keymsg.Alt {
					var populated bool = true
					for _, kti := range em.orderedKTIs { // check all required fields are populated
						if em.cfg[kti.key].Required && strings.TrimSpace(kti.ti.Value()) == "" {
							em.inputErr = kti.key + " is required"
							populated = false
							break
						}
					}
					if populated {
						if invalMsg, err := upd(em.orderedKTIs, &em.selectedData); err != nil {
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
		cmds := make([]tea.Cmd, len(em.orderedKTIs))
		for i, tti := range em.orderedKTIs {
			em.orderedKTIs[i].ti, cmds[i] = tti.ti.Update(msg)
		}
		cmd = tea.Batch(cmds...)
	}

	return cmd
}

// Blur existing TI, select and focus previous (higher) TI
func (em *editModel) previousTI() {
	em.orderedKTIs[em.tiIndex].ti.Blur()
	em.tiIndex -= 1
	if em.tiIndex < 0 {
		em.tiIndex = em.tiCount - 1
	}
	em.orderedKTIs[em.tiIndex].ti.Focus()
}

// Blur existing TI, select and focus next (lower) TI
func (em *editModel) nextTI() {
	em.orderedKTIs[em.tiIndex].ti.Blur()
	em.tiIndex += 1
	if em.tiIndex >= em.tiCount {
		em.tiIndex = 0
	}
	em.orderedKTIs[em.tiIndex].ti.Focus()
}

type Field struct {
	Required      bool   // is this field required to be populated?
	Title         string // field name displayed next to prompt and as flage name
	Usage         string // OPTIONAL. Flag usage displayed via -h
	FlagName      string // OPTIONAL. Defaults to DeriveFlagName() result.
	FlagShorthand rune   // OPTIONAL. '-x' form of FlagName.
	Order         int    // OPTIONAL. Top-Down (highest to lowest) display order of this field.

	// OPTIONAL.
	// Called once, at program start to generate a TI instead of using a generalize newTI()
	CustomTIFuncInit func() textinput.Model

	value *string // value of this field, set by flag or TI
}

// Transmute generates the list of TIs using the provided Field configuration. Fields with changed
// flags have their default set to the flags values. Fields without are given to the implementor to
// manually populate their default values from data's fields (see translate()).
func transmuteStruct(data types.SearchMacro,
	fs pflag.FlagSet,
	cfg Config,
	translateFunc getFieldFunction) (
	[]keyedTI, error,
) {
	var orderedKTIs []keyedTI = make([]keyedTI, len(cfg))

	var i uint8 = 0

	for k, v := range cfg {
		// create the TI, using the custom creation function if defined
		var ti textinput.Model
		if v.CustomTIFuncInit != nil {
			ti = v.CustomTIFuncInit()
		} else {
			ti = stylesheet.NewTI("", v.Required)
		}

		if fs.Changed(v.FlagName) { // if this field's flag was changed, update its default value
			// fetch the value
			if x, err := fs.GetString(v.FlagName); err != nil {
				return nil, err
			} else {
				ti.SetValue(x)
			}
		} else {
			// if this flag was not set,
			// the implementor must map it to the corresponding struct field
			clilog.Writer.Debugf("field '%v' requires translation", k)
			if t, err := translateFunc(data, k); err != nil {
				return nil, err
			} else {
				ti.SetValue(t)
			}
		}

		// add the TI to the list
		orderedKTIs[i] = keyedTI{key: k, ti: ti}

		// iterate
		i += 1
	}

	// with TIs built, sort them by order
	slices.SortFunc(orderedKTIs, func(a, b keyedTI) int {
		return cfg[b.key].Order - cfg[a.key].Order
	})

	return orderedKTIs, nil
}

// Takes the populated TIs, validates their input, and updates the gravwell backend.
func upd(ttis []keyedTI, data *types.SearchMacro) (title, invalMsg string, err error) {
	// no need to nil check; all required fields are checked already

	// rebuild the struct for the update call
	for i, tti := range ttis {
		switch tti.key {
		case "name":
			data.Name = strings.ToUpper(tti.ti.Value()) // name must always be uppercase
			ttis[i].ti.SetValue(data.Name)              // update it in case we return invalid or err
			// validate
			if strings.Contains(data.Name, " ") {
				return "name may not contains spaces", nil
			}
		case "description":
			data.Description = tti.ti.Value()
		case "expansion":
			data.Expansion = tti.ti.Value()
		}
	}

	// submit the updated struct
	return "", connection.Client.UpdateMacro(*data)
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
		for _, kti := range em.orderedKTIs {
			// color the title appropriately
			if em.cfg[kti.key].Required {
				sb.WriteString(tiFieldRequiredSty.Render(kti.key + ": "))
			} else {
				sb.WriteString(tiFieldOptionalSty.Render(kti.key + ": "))
			}
			sb.WriteString(kti.ti.View() + "\n")
		}
		sb.WriteString(colorizer.SubmitString("alt+enter", em.inputErr, em.updateErr, em.width))
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
	em.fs = flags()
	if em.addtlFlagFunc != nil {
		aflags := em.addtlFlagFunc()
		em.fs.AddFlagSet(&aflags)
	}

	// selecting mode
	em.list = list.Model{}

	// editting mode
	em.orderedKTIs = nil
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

	// check for an explicit macro id
	if id, err := em.fs.GetUint64("id"); err != nil {
		return "", nil, err
	} else if em.fs.Changed("id") {
		if em.selectedData, err = em.getFunc(id); err != nil {
			// treat this as an invalid argument
			return fmt.Sprintf("failed to fetch macro by id (%v): %v", id, err), nil, nil
		}
		// we can jump directly to editting phase on start
		if err := em.enterEditMode(); err != nil {
			em.mode = quitting
			clilog.Writer.Errorf("%v", err)
			return "", nil, err
		}

		return "", nil, nil
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

// Triggers the edit model to enter editting mode.
// This transmutes selectedData, and otherwise prepares the TIs, in the process.
func (em *editModel) enterEditMode() error {
	clilog.Writer.Debugf("editting macro %v", em.selectedData.Name)

	// transmute the selected item into a series of TIs
	var err error
	if em.orderedKTIs, err = transmuteStruct(em.selectedData, em.fs,
		em.cfg, em.transFunc); err != nil {
		return err
	}
	em.tiCount = len(em.orderedKTIs)
	if em.tiCount < 1 {
		return errors.New("no TIs created by transmutation")
	}

	em.orderedKTIs[0].ti.Focus() // focus the first TI

	em.mode = editting
	return nil
}

//#endregion interactive mode (model) implementation

type macroItem struct {
	title, description string
}

var _ list.DefaultItem = macroItem{}

func (mi macroItem) FilterValue() string { return mi.title }
func (mi macroItem) Title() string       { return mi.title }
func (mi macroItem) Description() string { return mi.description }
