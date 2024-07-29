package scaffoldedit

/**
 * More on Design:
 * Edit is definitely the most complex of the scaffolds, requiring components of both Create
 * (arbitrary TIs) and Delete (list possible structs/items).
 * By virtue of passing around structs and ids, it was always going to require multiple generics.
 * However, the decision to not use reflection was made fairly early.
 * I figured that reflection is
 * 1) slow
 * 2) error-prone (needing to look up qualified field names given by the implementor)
 * 3) an added layer of complexity on top of the already-in-play generics
 * Thus, no reflection.
 * The side effect of this, of course, is that we need yet more functions from the implementor and a
 * couple of trivial get/sets to be able to operate on the struct we want to update.
 *
 * Not sharing the Field struct between edit and create was a conscious choice to allow them to be
 * updated independently as it is more coincidental that they are similar.
 */

import (
	"errors"
	"fmt"
	"gwcli/action"
	"gwcli/clilog"
	"gwcli/mother"
	"gwcli/stylesheet"
	"gwcli/stylesheet/colorizer"
	"gwcli/utilities/keymaps"
	"gwcli/utilities/scaffold"
	"gwcli/utilities/treeutils"
	"gwcli/utilities/uniques"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/exp/constraints"
)

const (
	listHeightMax  = 40 // lines
	successStringF = "Successfully updated %v %v"
)
const ( // local flag values
	flagIDName   = "id"
	flagIDUsageF = "id of the %v to edit"
)

type id_t interface {
	constraints.Integer | uuid.UUID
}

// #region local styles

var (
	// TI field marked as required
	tiFieldRequiredSty = stylesheet.Header1Style
	// TI field marked as optional
	tiFieldOptionalSty = stylesheet.Header2Style
)

// #endregion

func NewEditAction[I id_t, S any](singular, plural string, cfg Config, funcs SubroutineSet[I, S]) action.Pair {
	funcs.guarantee() // check that all functions are given
	if len(cfg) < 1 { // check that config has fields in it
		panic("cannot edit with no fields defined")
	}

	var fs pflag.FlagSet = generateFlagSet(cfg, singular)

	cmd := treeutils.NewActionCommand(
		"edit",                             // use
		"edit a "+singular,                 // short
		"edit/alter an existing "+singular, // long
		[]string{"e"},                      // aliases
		func(cmd *cobra.Command, args []string) {
			var err error
			// hard branch on script mode
			var script bool
			if script, err = cmd.Flags().GetBool("script"); err != nil {
				clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error()+"\n")
				return
			}
			if script {
				runNonInteractive[I, S](cmd, cfg, funcs, singular)
			} else {
				runInteractive(cmd, args)
			}
		})

	// attach flags to cmd
	cmd.Flags().AddFlagSet(&fs)

	return treeutils.GenerateAction(cmd,
		newEditModel(cfg, singular, plural, funcs, fs),
	)
}

// Generates a flagset from the given configuration and appends flags native to scaffoldedit.
func generateFlagSet(cfg Config, singular string) pflag.FlagSet {
	var fs pflag.FlagSet
	for _, field := range cfg {
		if field.FlagName == "" {
			field.FlagName = uniques.DeriveFlagName(field.Title)
		}

		// map fields to their flags
		if field.FlagShorthand != 0 {
			fs.StringP(field.FlagName, string(field.FlagShorthand), "", field.Usage)
		} else {
			fs.String(field.FlagName, "", field.Usage)
		}
	}

	// attach native flags
	fs.StringP(flagIDName, "i", "", fmt.Sprintf(flagIDUsageF, singular))

	return fs
}

// run helper function.
// runNonInteractive is the --script portion of edit's runFunc.
// It requires --id be set and is ineffectual if an addtl/field flag was no given.
// Prints and error handles on its own; the program is expected to exit on its compeltion.
func runNonInteractive[I id_t, S any](cmd *cobra.Command, cfg Config, funcs SubroutineSet[I, S], singular string) {
	var err error
	var (
		id   I
		zero I
		itm  S
	)
	if strid, err := cmd.Flags().GetString(flagIDName); err != nil {
		clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error()+"\n")
		return
	} else {
		id, err = scaffold.FromString[I](strid)
		if err != nil {
			clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error()+"\n")
			return
		}
	}
	if id == zero { // id was not given
		fmt.Fprintln(cmd.OutOrStdout(), "--id is required in script mode")
		return
	}

	// get the item to edit
	if itm, err = funcs.SelectSub(id); err != nil {
		clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error()+"\n")
		return
	}

	var fieldUpdated bool   // was a value actually changed?
	for k, v := range cfg { // check each field for updates to be made
		// get current value
		curVal, err := funcs.GetFieldSub(itm, k)
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
			if inv, err := funcs.SetFieldSub(&itm, k, newVal); err != nil {
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
	identifier, err := funcs.UpdateSub(&itm)
	if err != nil {
		clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error()+"\n")
		return
	}
	fmt.Fprintf(cmd.OutOrStdout(), successStringF+"\n", singular, identifier)

}

// run helper function.
// Prints and error handles on its own; the program is expected to exit on its compeltion.
func runInteractive(cmd *cobra.Command, args []string) {
	// we have no way of knowing if the user has passed enough data to make the edit autonomously
	// ex: they provided one flag, but are they only planning to edit one flag?
	// therefore, just spawn mother; she is smart enough to handle the flags naturally
	if err := mother.Spawn(cmd.Root(), cmd, args); err != nil {
		clilog.Writer.Critical(err.Error())
	}
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

type editModel[I id_t, S any] struct {
	mode             mode                // current program state
	fs               pflag.FlagSet       // current state of the flagset
	singular, plural string              // forms of the noun
	width, height    int                 // tty dimensions, queried by SetArgs()
	funcs            SubroutineSet[I, S] // functions provided by implementor

	cfg Config // RO configuration provided by the caller

	data []S // data retrieved by fchFunc

	// selecting mode
	list            list.Model // list displayed during `selecting` mode
	listInitialized bool

	// editting mode
	orderedKTIs  []keyedTI // TIs will be displayed in array order
	tiIndex      int       // array index of active TI
	tiCount      int       // len(ttis)
	selectedData S         // item chosen from the list
	inputErr     string    // input is erroneous
	updateErr    string    // error occured performing the update
}

// Creates and returns a new edit model, ready for intreactive use.
func newEditModel[I id_t, S any](cfg Config, singular, plural string,
	funcs SubroutineSet[I, S], initialFS pflag.FlagSet) *editModel[I, S] {
	em := &editModel[I, S]{
		mode:     idle,
		fs:       initialFS,
		singular: singular,
		plural:   plural,
		cfg:      cfg,
		funcs:    funcs,
	}

	return em
}

func (em *editModel[I, S]) SetArgs(_ *pflag.FlagSet, tokens []string) (
	invalid string, onStart tea.Cmd, err error,
) {
	// parse the flags, save them for later, when TIs are created
	if err := em.fs.Parse(tokens); err != nil {
		return "", nil, err
	}

	// check for an explicit ID
	if em.fs.Changed("id") {
		var id I
		if strid, err := em.fs.GetString(flagIDName); err != nil {
			return "", nil, err
		} else {
			id, err = scaffold.FromString[I](strid)
			if err != nil {
				return "failed to parse id from " + strid, nil, nil
			}
		}

		// select the item associated to the id
		if em.selectedData, err = em.funcs.SelectSub(id); err != nil {
			// treat this as an invalid argument
			return fmt.Sprintf("failed to fetch %s by id (%v): %v", em.singular, id, err), nil, nil
		}
		// we can jump directly to editting phase on start
		if err := em.enterEditMode(); err != nil {
			em.mode = quitting
			clilog.Writer.Errorf("%v", err)
			return "", nil, err
		}

		return "", nil, nil

	}

	// fetch edit-able items
	if em.data, err = em.funcs.FetchSub(); err != nil {
		return
	}

	var dataCount = len(em.data)

	// check for a lack of data
	if dataCount < 1 { // die
		em.mode = quitting
		return "", tea.Printf("You have no %v that can be editted", em.plural), nil
	}

	// transmute data into list items
	var itms []list.Item = make([]list.Item, dataCount)
	for i, s := range em.data {
		itms[i] = listItem{
			title:       em.funcs.GetTitleSub(s),
			description: em.funcs.GetDescriptionSub(s),
		}
	}

	// generatelist
	em.list = list.New(itms, list.NewDefaultDelegate(), 80, listHeightMax)
	em.list.KeyMap = keymaps.ListKeyMap()
	em.listInitialized = true
	em.mode = selecting

	return "", uniques.FetchWindowSize, nil
}

func (em *editModel[I, S]) Update(msg tea.Msg) tea.Cmd {
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		em.width = msg.Width
		em.height = msg.Height
		// if we skipped directly to edit mode, list will be nil
		if em.listInitialized {
			em.list.SetSize(em.width, min(msg.Height-2, listHeightMax))
		}
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
func (em *editModel[I, S]) updateSelecting(msg tea.Msg) tea.Cmd {
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

func (em *editModel[I, S]) updateEditting(msg tea.Msg) tea.Cmd {
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
					if inv, err := em.funcs.SetFieldSub(&em.selectedData, kti.key, kti.ti.Value()); err != nil {
						em.mode = quitting
						return tea.Println(err, "\n", "no changes made")
					} else if inv != "" {
						em.inputErr = inv
						return textinput.Blink
					}
				}

				// perform the update
				identifier, err := em.funcs.UpdateSub(&em.selectedData)
				if err != nil {
					em.updateErr = err.Error()
					return textinput.Blink
				}
				// success
				em.mode = quitting
				return tea.Printf(successStringF, em.singular, identifier)
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
func (em *editModel[I, S]) previousTI() {
	em.orderedKTIs[em.tiIndex].ti.Blur()
	em.tiIndex -= 1
	if em.tiIndex < 0 {
		em.tiIndex = em.tiCount - 1
	}
	em.orderedKTIs[em.tiIndex].ti.Focus()
}

// Blur existing TI, select and focus next (lower) TI
func (em *editModel[I, S]) nextTI() {
	em.orderedKTIs[em.tiIndex].ti.Blur()
	em.tiIndex += 1
	if em.tiIndex >= em.tiCount {
		em.tiIndex = 0
	}
	em.orderedKTIs[em.tiIndex].ti.Focus()
}

func (em *editModel[I, S]) View() string {
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

func (em *editModel[I, S]) Done() bool {
	return em.mode == quitting
}

func (em *editModel[I, S]) Reset() error {
	var zero S

	em.mode = idle
	em.data = nil
	em.fs = generateFlagSet(em.cfg, em.singular)

	// selecting mode
	em.list = list.Model{}
	em.listInitialized = false

	// editting mode
	em.orderedKTIs = nil
	em.tiIndex = 0
	em.tiCount = 0
	em.selectedData = zero
	em.inputErr = ""
	em.updateErr = ""

	return nil
}

// Triggers the edit model to enter editting mode, performing all required data setup.
func (em *editModel[I, S]) enterEditMode() error {
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
			curVal, err := em.funcs.GetFieldSub(em.selectedData, k)
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

type listItem struct {
	title, description string
}

var _ list.DefaultItem = listItem{}

func (mi listItem) FilterValue() string { return mi.title }
func (mi listItem) Title() string       { return mi.title }
func (mi listItem) Description() string { return mi.description }
