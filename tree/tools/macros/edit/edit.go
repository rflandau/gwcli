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

const (
	listHeightMax  = 40 // lines
	successStringF = "Successfully updated %v %v "
)

//#region function signatures

// function to get the specific, edit-able struct and skip list/selecting mode.
type selectFunction = func(id uint64) (
	item types.SearchMacro, err error,
)

// function to fetch all edit-able structs
type fetchAllFunction = func() (
	items []types.SearchMacro, err error,
)

// Function to retrieve the struct value associated to the field key without reflection.
// This is probably a switch statement that maps (key -> item.X).
//
// Sister to setFieldFunction.
type getFieldFunction = func(item types.SearchMacro, fieldKey string) (
	value string, err error,
)

// Function to set the struct value associated to the field key without reflection.
// This is probably a switch statement that maps (key -> item.X).
// Returns invalid if the value is invalid for the keyed field and err on an unrecoverable error.
//
// Sister to getFieldFunction.
type setFieldFunction = func(item *types.SearchMacro, fieldKey, val string) (
	invalid string, err error,
)

// function to perform the actual update of the data on the GW instance
type updateStructFunction = func(data *types.SearchMacro) (
	identifier string, err error,
)

// Set of all functions, to make it easier to pass them around internally.
// All fields are required.
type functionSet struct {
	sel  selectFunction       // fetch a specific editable struct
	fch  fetchAllFunction     // used in interactive mode to fetch all editable structs
	getF getFieldFunction     // get a value within the struct
	setF setFieldFunction     // set a value within the struct
	upd  updateStructFunction // submit the struct as updated
}

// Validates that all functions were set.
// Panics if any are missing.
func (funcs *functionSet) guarantee() {
	if funcs.sel == nil {
		panic("select function is required")
	}
	if funcs.fch == nil {
		panic("fetch all function is required")
	}
	if funcs.getF == nil {
		panic("get field function is required")
	}
	if funcs.setF == nil {
		panic("set field function is required")
	}
	if funcs.upd == nil {
		panic("update struct function is required")
	}
}

//#endregion

type Config = map[string]*Field

// #region local styles
var (
	tiFieldRequiredSty = stylesheet.Header1Style
	tiFieldOptionalSty = stylesheet.Header2Style
)

// #endregion

func NewMacroEditAction() action.Pair {
	// TODO replace these with parameters
	cfg := Config{
		"name": &Field{
			Required: true,
			Title:    "Name",
			Usage:    stylesheet.FlagDescMacroName,
			FlagName: uniques.DeriveFlagName("name"),
			Order:    100,
		},
		"description": &Field{
			Required: true,
			Title:    "Description",
			Usage:    stylesheet.FlagDescMacroDesc,
			FlagName: uniques.DeriveFlagName("description"),
			Order:    80,
		},
		"expansion": &Field{
			Required: true,
			Title:    "Expansion",
			Usage:    stylesheet.FlagDescMacroExpansion,
			FlagName: uniques.DeriveFlagName("expansion"),
			Order:    60,
		},
	}
	funcs := functionSet{
		sel: getMacro,
		fch: func() ([]types.SearchMacro, error) {
			return connection.Client.GetUserMacros(connection.MyInfo.UID)
		},
		getF: func(item types.SearchMacro, fieldKey string) (string, error) {
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
		setF: func(item *types.SearchMacro, fieldKey, val string) (string, error) {
			switch fieldKey {
			case "name":
				if strings.Contains(val, " ") {
					return "name may not contain spaces", nil
				}
				val = strings.ToUpper(val)
				item.Name = val
			case "description":
				item.Description = val
			case "expansion":
				item.Expansion = val
			default:
				return "", fmt.Errorf("unknown field key: %v", fieldKey)
			}
			return "", nil
		},
		upd: func(data *types.SearchMacro) (identifier string, err error) {
			if err := connection.Client.UpdateMacro(*data); err != nil {
				return "", err
			}
			return data.Name, nil
		},
	}
	funcs.guarantee() // check that all functions are given
	if len(cfg) < 1 { // check that config has fields in it
		panic("cannot edit with no fields defined")
	}

	cmd := treeutils.NewActionCommand("edit", "edit a macro", "edit/alter an existing macro",
		[]string{"e"}, func(c *cobra.Command, s []string) {})

	cmd.Run = func(cmd *cobra.Command, args []string) {
		var err error
		// hard branch on script mode
		var script bool
		if script, err = cmd.Flags().GetBool("script"); err != nil {
			clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error()+"\n")
			return
		}
		if script {
			runNonInteractive(cmd, cfg, funcs)
		}
	}

	// assign base flags
	flags, aflags := flags(), addtlFlags()
	cmd.Flags().AddFlagSet(&flags)
	cmd.Flags().AddFlagSet(&aflags)

	return treeutils.GenerateAction(cmd, newEditModel(
		cfg,
		funcs,
		addtlFlags))
}

const ( // local flag names
	flagID = "id"
)

// run helper function
// runNonInteractive is the --script portion of edit's runFunc.
// It requires --id be set and is ineffectual if an addtl/field flag was no given.
// Prints and error handles on its own; the program is expected to exit on its compeltion.
func runNonInteractive(cmd *cobra.Command, cfg Config, funcs functionSet) {
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

	// get the macro to edit
	if itm, err = funcs.sel(id); err != nil {
		clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error()+"\n")
		return
	}

	var fieldUpdated bool   // was a value actually changed?
	for k, v := range cfg { // check each field for updates to be made
		// get current value
		curVal, err := funcs.getF(itm, k)
		if err != nil {
			clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error()+"\n")
			return
		}
		var newVal string = curVal
		if cmd.Flags().Changed(v.FlagName) { // flag *presumably* updates the field
			if x, err := cmd.Flags().GetString(v.FlagName); err != nil {
				clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error()+"\n")
				return
			} else {
				newVal = x
			}
		}

		if newVal != curVal { // update the struct
			fieldUpdated = true // note if a change occured
			if inv, err := funcs.setF(&itm, k, newVal); err != nil {
				clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error()+"\n")
				return
			} else if inv != "" {
				fmt.Fprintln(cmd.OutOrStdout(), inv)
				return
			}
		}
	}

	if !fieldUpdated { // only bother to update if at least one field was changed
		clilog.Tee(clilog.INFO, cmd.OutOrStdout(), "no field would be updated; quitting...\n")
		return
	}

	// perform the actual update
	identifier, err := funcs.upd(&itm)
	if err != nil {
		clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error()+"\n")
		return
	}
	fmt.Fprintf(cmd.OutOrStdout(), successStringF+"\n", "macro", identifier)

}

// base flagset always available to edit actions
func flags() pflag.FlagSet {
	fs := pflag.FlagSet{}
	fs.Uint64(flagID, 0, "id of the macro to edit")
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
	width, height int                  // tty dimensions, queried by SetArgs()
	funcs         functionSet          // functions provided by implementor

	cfg Config // RO configuration provided by the caller

	data []types.SearchMacro // data retrieved by fchFunc

	// selecting mode
	list list.Model // list displayed during `selecting` mode

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
	funcs functionSet,
	addtlFlagFunc func() pflag.FlagSet) *editModel {
	// sanity check required arguments
	if cfg == nil {
		panic("Configuration cannot be nil")
	}

	em := &editModel{
		mode:          idle,
		cfg:           cfg,
		funcs:         funcs,
		addtlFlagFunc: addtlFlagFunc,
	}
	em.fs = flags()
	if em.addtlFlagFunc != nil {
		aflags := em.addtlFlagFunc()
		em.fs.AddFlagSet(&aflags)
	}

	return em
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
		if em.selectedData, err = em.funcs.sel(id); err != nil {
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
	if em.data, err = em.funcs.fch(); err != nil {
		return
	}

	var dataCount = len(em.data)

	// check for a lack of data
	if dataCount < 1 { // die
		em.mode = quitting
		return "", tea.Printf("You have no %v that can be editted", "macros"), nil
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

func (em *editModel) Update(msg tea.Msg) tea.Cmd {
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		em.width = msg.Width
		em.height = msg.Height
		em.list.SetSize(em.width, min(msg.Height-2, listHeightMax))
	}

	// switch handling based on mode
	switch em.mode {
	case quitting:
		return nil
	case selecting:
		return em.updateSelecting(msg)
	case editting:
		return em.updateEditting(msg)
	default:
		clilog.Writer.Criticalf("unknown edit mode %v.", em.mode)
		clilog.Writer.Debugf("model dump: %#v.", em)
		clilog.Writer.Info("Returning control to Mother...")
		em.mode = quitting
		return textinput.Blink
	}
}

// Update() handling for selecting mode.
// Updates the list and transitions to editting mode if an item is selected.
func (em *editModel) updateSelecting(msg tea.Msg) tea.Cmd {
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
	var cmd tea.Cmd
	em.list, cmd = em.list.Update(msg)
	return cmd
}

func (em *editModel) updateEditting(msg tea.Msg) tea.Cmd {
	if keymsg, ok := msg.(tea.KeyMsg); ok {
		em.inputErr = "" // clear input errors on new key input
		switch keymsg.Type {
		case tea.KeyEnter:
			if keymsg.Alt { // check for a submission via alt+enter
				em.updateErr = "" // clear existing updateErr

				var missing []string
				for _, kti := range em.orderedKTIs { // check all required fields are populated
					if em.cfg[kti.key].Required && strings.TrimSpace(kti.ti.Value()) == "" {
						missing = append(missing, kti.key)
					}
				}

				// if fields are missing, warn and do not submit
				if len(missing) > 0 {
					imploded := strings.Join(missing, ", ")
					copula := "is"
					if len(missing) > 1 {
						copula = "are"
					}
					em.inputErr = fmt.Sprintf("%v %v required", imploded, copula)
					return textinput.Blink
				}

				for _, kti := range em.orderedKTIs {
					// yank the TI values and reinstall them into a data structure to update against
					if inv, err := em.funcs.setF(&em.selectedData, kti.key, kti.ti.Value()); err != nil {
						em.mode = quitting
						return tea.Println(err, "\n", "no changes made")
					} else if inv != "" {
						em.inputErr = inv
						return textinput.Blink
					}
				}

				// perform the update
				identifier, err := em.funcs.upd(&em.selectedData)
				if err != nil {
					em.updateErr = err.Error()
					return textinput.Blink
				}
				// success
				em.mode = quitting
				return tea.Printf(successStringF, "macro", identifier)
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
	return tea.Batch(cmds...)
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
				Render("Press space or enter to select")
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

// Triggers the edit model to enter editting mode, performing all required data setup.
func (em *editModel) enterEditMode() error {
	clilog.Writer.Debugf("editting macro %v", em.selectedData.Name)

	// prepare list
	em.orderedKTIs = make([]keyedTI, len(em.cfg))

	// use the get function to pull current values for each field and display them in their
	// respective TIs
	var i uint8 = 0
	for k, field := range em.cfg {
		// create the ti
		var ti textinput.Model
		if field.CustomTIFuncInit != nil {
			ti = field.CustomTIFuncInit()
		} else {
			ti = stylesheet.NewTI("", field.Required)
		}

		var setByFlag bool
		if em.fs.Changed(field.FlagName) { // prefer flag value
			if x, err := em.fs.GetString(field.FlagName); err == nil {
				ti.SetValue(x)
				setByFlag = true
			}
		}

		if !setByFlag { // fallback to current value
			curVal, err := em.funcs.getF(em.selectedData, k)
			if err != nil {
				return err
			}
			ti.SetValue(curVal)
		}

		// attach TI to list
		em.orderedKTIs[i] = keyedTI{key: k, ti: ti}
		i += 1
	}

	em.tiCount = len(em.orderedKTIs)
	if em.tiCount < 1 {
		return errors.New("no TIs created by transmutation")
	}

	// order TIs from highest to lowest orders
	slices.SortFunc(em.orderedKTIs, func(a, b keyedTI) int {
		return em.cfg[b.key].Order - em.cfg[a.key].Order
	})

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
