/*
A create action creates a shallow list of inputs for the user to fill via flags or interactive
TIs before being passed back to the progenitor to transform into usable data for their create
function.

The available fields are fairly configurable, the progentior provides their own map of Field
structs, and easily extensible, the struct can have more options or formats bolted on without too
much trouble.

This scaffold is a bit easier to extend than Delete and List, given it did not require generics.

Look to the scheduled query creation action (external to the one built into DataScope) or macro
creation action as two examples of implementation styles.

! Once a Config is given by the caller, it should be considered ReadOnly.

NOTE: More complex creation with nested options and mutli-stage flows should be built
independently. This scaffold is intended for simple, handful-of-field creations.

Example implementation:

	func NewCreateAction() action.Pair {
		n := scaffoldcreate.NewField(true, "name", 100)
		d := scaffoldcreate.NewField(true, "value", 90)
		fields := scaffoldcreate.Config{
			"name":  n,
			"value": d,
			"field3": scaffoldcreate.Field{
				Required:      true,
				Title:         "field3",
				Usage:         "field 3 usage",
				Type:          scaffoldcreate.Text,
				FlagName:      "flagn",
				FlagShorthand: 'f',
				DefaultValue:  "",
				TI: struct {
					Order       int
					Placeholder string
					Validator   func(s string) error
				}{
					Order: 80,
				},
			},
		}

		return scaffoldcreate.NewCreateAction("singular noun", fields, create)
	}

	func create(_ scaffoldcreate.Config, vals scaffoldcreate.Values) (any, string, error) {
		id, err := connection.Client.X()
		return id, "", err
	}
*/
package scaffoldcreate

import (
	"fmt"
	"gwcli/action"
	"gwcli/clilog"
	"gwcli/mother"
	"gwcli/stylesheet"
	"gwcli/stylesheet/colorizer"
	"gwcli/treeutils"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	errMissingRequiredFlags = "missing required flags %v"
	createdSuccessfully     = "Successfully created %v (ID: %v)."
)

// #region local styles
var (
	tiFieldRequiredSty = stylesheet.Header1Style
	tiFieldOptionalSty = stylesheet.Header2Style
)

//#endregion

// keys -> Field; used as (ReadOnly) configuration for this creation instance
type Config = map[string]Field

type Values = map[string]string

// signature the supplied creation function must match
type CreateFunc func(cfg Config, values Values) (id any, invalid string, err error)

func NewCreateAction(singular string,
	fields Config,
	create CreateFunc) action.Pair {
	// pull flags from provided fields
	//const mappedString = "mapped field %v (key: %v) to %v flag %v"
	var flags pflag.FlagSet = installFlagsFromFields(fields)

	cmd := treeutils.NewActionCommand(
		"create",                 // use
		"create a "+singular,     // short
		"create a new "+singular, // long
		[]string{},               // aliases
		func(c *cobra.Command, s []string) {
			// get standard flags
			script, err := c.Flags().GetBool("script")
			if err != nil {
				clilog.Tee(clilog.ERROR, c.ErrOrStderr(), err.Error()+"\n")
				return
			}
			// get field flags
			var values map[string]string
			if vals, mr, err := getValuesFromFlags(c.Flags(), fields); err != nil {
				clilog.Tee(clilog.ERROR, c.ErrOrStderr(), err.Error()+"\n")
				return
			} else if mr != nil {
				if !script {
					if err := mother.Spawn(c.Root(), c, s); err != nil {
						clilog.Writer.Critical(err.Error())
					}
					return
				} else {
					fmt.Fprintf(c.OutOrStdout(), errMissingRequiredFlags+"\n", mr)
				}
				return
			} else {
				values = vals
			}

			// attempt to create the new X
			if id, inv, err := create(fields, values); err != nil {
				clilog.Tee(clilog.ERROR, c.ErrOrStderr(), err.Error()+"\n")
				return
			} else if inv != "" { // some of the flags were invalid
				fmt.Fprintln(c.OutOrStdout(), inv)
				return
			} else {
				fmt.Fprintf(c.OutOrStdout(), "Successfully created %v (ID: %v).", singular, id)
			}
		})

	// attach mined flags to cmd
	cmd.Flags().AddFlagSet(&flags)

	return treeutils.GenerateAction(cmd, newCreateModel(fields, singular, create))
}

// Given a parsed flagset and the field configuration, builds a corollary map of field values.
//
// Returns the values for each flag (default if unset), a list of required fields (as their flag
// names) that were not set, and an error (if one occurred).
func getValuesFromFlags(fs *pflag.FlagSet, fields Config) (
	values Values, missingRequireds []string, err error,
) {
	values = make(Values)
	for k, f := range fields {
		switch f.Type {
		case Text:

			flagVal, err := fs.GetString(f.FlagName)
			if err != nil {
				return nil, nil, err
			}
			// if this value is required, but unset, add it to the list
			if f.Required && !fs.Changed(f.FlagName) {
				missingRequireds = append(missingRequireds, f.FlagName)
			}

			values[k] = flagVal
		default:
			panic("developer error: unknown field type: " + f.Type)
		}
	}
	return values, missingRequireds, nil
}

//#region interactive mode (model) implementation

const defaultWidth = 80 // default wrap width, used before initial WinMsgSz arrives

type mode uint // state of the interactive application

const (
	inputting mode = iota // user entering data
	quitting              // done
)

// a tuple for storing a TI and the field key it is associated with
type keyedTI struct {
	key string
	ti  textinput.Model
}

// interactive model that builds out inputs based on the read-only Config supplied on creation.
type createModel struct {
	mode mode

	width int // tty width

	singular string // "macro", "search", etc

	fields Config // RO configuration provided by the caller

	orderedTIs []keyedTI // Ordered array of map keys, based on Config.TI.Order
	selected   uint      // currently focused ti (in key order index)

	inputErr  string // the reason inputs are invalid
	createErr string // the reason the last create failed (not for invalid parameters)

	fs pflag.FlagSet // parsed flag values, mined from the Config
	cf CreateFunc    // function to create the new entity
}

// Creates and returns a create Model, ready for interactive usage via Mother.
func newCreateModel(fields Config, singular string, cf CreateFunc) *createModel {
	c := &createModel{
		mode:       inputting,
		width:      defaultWidth,
		singular:   singular,
		fields:     fields,
		orderedTIs: make([]keyedTI, 0),
		fs:         installFlagsFromFields(fields),
		cf:         cf,
	}

	for k, f := range fields {
		// generate the TI
		kti := keyedTI{
			key: k,
			ti:  stylesheet.NewTI(f.DefaultValue, !f.Required),
		}
		kti.ti.Validate = f.TI.Validator
		if f.TI.Placeholder != "" {
			kti.ti.Placeholder = f.TI.Placeholder
		}
		c.orderedTIs = append(c.orderedTIs, kti)
	}
	// sort keys from highest order to lowest order
	slices.SortFunc(c.orderedTIs, func(a, b keyedTI) int {
		return fields[b.key].TI.Order - fields[a.key].TI.Order
	})

	if len(c.orderedTIs) > 0 {
		c.orderedTIs[0].ti.Focus()
	}

	return c
}

func (c *createModel) Update(msg tea.Msg) tea.Cmd {
	if c.mode == quitting {
		return nil
	}
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		c.inputErr = "" // clear last input error
		switch keyMsg.Type {
		case tea.KeyUp, tea.KeyShiftTab:
			c.focusPrevious()
			return textinput.Blink
		case tea.KeyDown, tea.KeyTab:
			c.focusNext()
			return textinput.Blink
		case tea.KeyEnter:
			if keyMsg.Alt { // only submit on alt+enter
				c.createErr = "" // clear last error
				// extract values from TIs
				values, mr := c.extractValuesFromTIs()
				if mr != nil {
					c.inputErr = fmt.Sprintf("%v are required", mr)
					return nil
				}
				id, invalid, err := c.cf(c.fields, values)
				if err != nil {
					c.createErr = err.Error()
					return nil
				} else if invalid != "" {
					c.inputErr = invalid
					return nil
				}
				// done, die
				c.mode = quitting
				return tea.Println(fmt.Sprintf(createdSuccessfully, c.singular, id))
			}
		}
	} else if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
		c.width = sizeMsg.Width
		return nil
	}
	// pass message to currently focused ti
	var cmd tea.Cmd
	c.orderedTIs[c.selected].ti, cmd = c.orderedTIs[c.selected].ti.Update(msg)
	return cmd
}

// Blurs the current ti, selects and focuses the next (indexically) one.
func (c *createModel) focusNext() {
	c.orderedTIs[c.selected].ti.Blur()
	c.selected += 1
	if c.selected >= uint(len(c.orderedTIs)) { // jump to start
		c.selected = 0
	}
	c.orderedTIs[c.selected].ti.Focus()
}

// Blurs the current ti, selects and focuses the previous (indexically) one.
func (c *createModel) focusPrevious() {
	c.orderedTIs[c.selected].ti.Blur()
	if c.selected == 0 { // jump to end
		c.selected = uint(len(c.orderedTIs)) - 1
	} else {
		c.selected -= 1
	}
	c.orderedTIs[c.selected].ti.Focus()
}

// Generates the corrollary value map from the TIs.
//
// Returns the values for each TI (mapped to their Config key), a list of required fields (as their
// field.Title names) that were not set, and an error (if one occured).
func (c *createModel) extractValuesFromTIs() (
	values Values, missingRequireds []string,
) {
	values = make(Values)
	for _, kti := range c.orderedTIs {
		val := strings.TrimSpace(kti.ti.Value())
		field := c.fields[kti.key]
		if val == "" && field.Required {
			missingRequireds = append(missingRequireds, field.Title)
		}

		values[kti.key] = val
	}

	return values, missingRequireds
}

// Iterates through the keymap, drawing each ti and title in key key order
func (c *createModel) View() string {
	var sb strings.Builder

	for i, kti := range c.orderedTIs {
		var title string
		// color the title appropriately
		if c.fields[kti.key].Required {
			title = tiFieldRequiredSty.Render(c.fields[kti.key].Title + ": ")
		} else {
			title = tiFieldOptionalSty.Render(c.fields[kti.key].Title + ": ")
		}
		sb.WriteString(title)

		// if window width is too small, bump TI to next line
		if c.width <= (lipgloss.Width(title) + c.orderedTIs[i].ti.Width) { // include equals for a 1 cell buffer
			sb.WriteString("\n")
		}
		sb.WriteString(c.orderedTIs[i].ti.View() + "\n")
	}

	// display errors, if they exist
	// note: result will always be an error string, as we exit on success
	sb.WriteString(colorizer.SubmitString("alt+enter", c.inputErr, c.createErr, c.width))

	return sb.String()
}

func (c *createModel) Done() bool {
	return c.mode == quitting
}

func (c *createModel) Reset() error {
	c.mode = inputting

	var wg sync.WaitGroup
	wg.Add(2)
	// reset TIs
	go func() {
		for i := range c.orderedTIs {
			c.orderedTIs[i].ti.Reset()
			c.orderedTIs[i].ti.Blur()
		}
		wg.Done()
	}()
	// refresh flags to their original, unparsed and unvalued state
	go func() { c.fs = installFlagsFromFields(c.fields); wg.Done() }()

	wg.Wait()

	c.createErr = ""
	c.inputErr = ""
	c.selected = 0
	if len(c.orderedTIs) > 0 {
		c.orderedTIs[0].ti.Focus()
	}
	return nil
}

func (c *createModel) SetArgs(_ *pflag.FlagSet, tokens []string) (
	invalid string, onStart []tea.Cmd, err error,
) {
	if err := c.fs.Parse(tokens); err != nil {
		return "", nil, err
	}

	// we do not need to check missing requires when run from mother
	if flagVals, _, err := getValuesFromFlags(&c.fs, c.fields); err != nil {
		return "", nil, err
	} else {
		// set flag values as the starter values in their corresponding TI
		for i, kti := range c.orderedTIs {
			c.orderedTIs[i].ti.SetValue(flagVals[kti.key])
		}
	}

	return "", []tea.Cmd{fetchWindowSize}, nil
}

// Queries for avaialble window size so we can wrap text by width, once this arrives.
// NOTE: width and height are returned, but create only uses width.
func fetchWindowSize() tea.Msg {
	w, h, err := term.GetSize(os.Stdin.Fd())
	if err != nil {
		clilog.Writer.Errorf("Failed to fetch terminal size: %v", err)
	}
	return tea.WindowSizeMsg{Width: w, Height: h}
}

//#endregion
