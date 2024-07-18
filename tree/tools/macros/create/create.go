package create

import (
	"fmt"
	"gwcli/action"
	"gwcli/clilog"
	"gwcli/connection"
	"gwcli/mother"
	"gwcli/stylesheet"
	"gwcli/treeutils"
	"strings"
	"unicode"

	"github.com/gravwell/gravwell/v3/client/types"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const ( // flag names
	flagName = "name"
	flagDesc = "description"
	flagExp  = "expansion"
)

func NewMacroCreateAction() action.Pair {
	// create the action
	cmd := treeutils.NewActionCommand("create", "create a new macro", "", []string{}, run)

	// establish local flags
	fs := flags()

	cmd.Flags().AddFlagSet(&fs)

	return treeutils.GenerateAction(cmd, NewCreate())
}

func flags() pflag.FlagSet {
	fs := pflag.FlagSet{}

	fs.StringP(flagName, "n", "", "the shorthand that will be expanded")
	fs.StringP(flagDesc, "d", "", "(flavour) description")
	fs.StringP(flagExp, "e", "", "value for the macro to expand to")

	return fs
}

// Creates a macro with the given values anmd returns the id it was assigned
func createMacro(name, desc, value string) (uint64, error) {
	// via the web gui, adding a macro requies a name and value (plus optional desc)
	macro := types.SearchMacro{Name: name, Description: desc, Expansion: value}

	id, err := connection.Client.AddMacro(macro)
	if err != nil {
		clilog.Writer.Warnf("Failed to create Macro: %s", err.Error())
		return 0, err
	}

	return id, nil
}

//#region cobra command

func run(cmd *cobra.Command, s []string) {
	// check script mode
	script, err := cmd.Flags().GetBool("script")
	if err != nil {
		clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error()+"\n")
		return
	}

	// fetch data from flags
	name, err := cmd.Flags().GetString(flagName)
	if err != nil {
		clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error()+"\n")
		return
	}
	name = strings.ToUpper(name) // name must be caps

	desc, err := cmd.Flags().GetString(flagDesc)
	if err != nil {
		clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error()+"\n")
		return
	}
	value, err := cmd.Flags().GetString(flagExp)
	if err != nil {
		clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error()+"\n")
		return
	}

	// check required flags
	if name == "" || value == "" {
		if script { // fail out
			fmt.Fprintf(
				cmd.ErrOrStderr(), "--%v, --%v, --%v are required in script mode\n",
				flagName, flagDesc, flagExp,
			)
			return
		}
		// spin up mother
		if err := mother.Spawn(cmd.Root(), cmd, s); err != nil {
			clilog.Tee(clilog.CRITICAL, cmd.ErrOrStderr(),
				"failed to spawn a mother instance: "+err.Error())
		}
		return
	}

	if id, err := createMacro(name, desc, value); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Failed to create macro: %v\n", err.Error())
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Successfully created macro %v (ID: %v)\n", name, id)
	}
}

//#endregion

//#region actor implementation

type input int

const (
	name input = iota
	desc
	value
)

const (
	helpShowAllInitial = false // starting state of help.model.ShowAll
)

type create struct {
	done bool

	fs pflag.FlagSet

	focusedInput input
	help         struct {
		model help.Model
		keys  helpKeyMap
	}
	ti []textinput.Model // name, desc, value
}

func NewCreate() *create {
	c := &create{done: false}

	c.fs = flags()

	c.ti = make([]textinput.Model, 3)
	for i := 0; i < 3; i++ {
		c.ti[i] = textinput.New()
		c.ti[i].Width = stylesheet.TIWidth
	}
	// the first ti (name) requires extra initialization (focus and validation)
	c.ti[0].Validate = func(s string) error {
		if err := types.CheckMacroName(s); err != nil {
			return err
		}
		return nil
	}
	c.ti[0].Focus()
	c.focusedInput = name

	// set up help
	c.help.model = help.New()
	c.help.keys = helpKeyMap{
		Next: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next"),
		),
		Prev: key.NewBinding(
			key.WithKeys(tea.KeyShiftTab.String()),
			key.WithHelp(tea.KeyShiftTab.String(), "next"),
		),
		Help: key.NewBinding(
			key.WithKeys(tea.KeyF1.String(), tea.KeyCtrlH.String()),
			key.WithHelp(tea.KeyF1.String()+"/"+tea.KeyCtrlH.String(), "toggle help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "return to navigation"),
		),
	}
	c.help.model.ShowAll = helpShowAllInitial

	return c
}

func (c *create) Update(msg tea.Msg) tea.Cmd {

	if c.done {
		return nil
	}

	switch msg := msg.(type) { // check for meta inputs
	case tea.KeyMsg: // only KeyMsg could require special handling
		switch {
		case msg.Type == tea.KeyEnter:
			clilog.Writer.Debugf("Create.Update received enter %v", msg.String())
			if c.focusedInput == value &&
				c.ti[name].Value() != "" &&
				c.ti[desc].Value() != "" &&
				c.ti[value].Value() != "" { // if last input and inputs are populated, attempt to create the macros
				if id, err := createMacro(c.ti[name].Value(), c.ti[desc].Value(), c.ti[value].Value()); err != nil {
					c.Reset()
					return tea.Printf("Failed to create macro %v (ID: %v):%v\n", name, id, err)
				} else {
					c.done = true
					return tea.Printf("Successfully created macro %v (ID: %v)\n", name, id)
				}

			} else {
				c.focusNext()
			}

		case msg.Type == tea.KeyTab:
			c.focusNext()

		case msg.Type == tea.KeyShiftTab:
			c.focusPrevious()

		case key.Matches(msg, c.help.keys.Help):
			clilog.Writer.Debugf("Swapping showall")
			c.help.model.ShowAll = !c.help.model.ShowAll

		default:
			// other key messages getting passed to name need to be upper-cased
			// if passing text to name field, upper-case it
			if c.focusedInput == name {
				for i, r := range msg.Runes {
					if unicode.IsLetter(r) {
						msg.Runes[i] = unicode.ToUpper(r)
					}
				}
			}
		}
	}

	// update focused input
	var cmd tea.Cmd
	c.ti[c.focusedInput], cmd = c.ti[c.focusedInput].Update(msg)

	return cmd
}

func (c *create) View() string {
	fields := fmt.Sprintf("Name: %s\n"+
		"Desc: %s\n"+
		"Expansion: %s \n", c.ti[name].View(), c.ti[desc].View(), c.ti[value].View())

	helpDisplay := c.help.model.View(c.help.keys)

	return stylesheet.Composable.Focused.Render(fields) + "\n" + helpDisplay
}

func (c *create) Done() bool {
	return c.done
}

// Resets the model to its initial state, dropping all data. This ensures the
// next call is against a clean slate
func (c *create) Reset() error {
	for i := range c.ti {
		c.ti[i].Reset()
	}

	// pflag has no way to unset flag values,
	// so we need to establish a new localFlagset
	c.fs = flags()

	c.ti[c.focusedInput].Blur()
	c.focusedInput = name
	c.ti[c.focusedInput].Focus()

	c.done = false
	c.help.model.ShowAll = helpShowAllInitial
	return nil
}

// focusNext determines and focuses the following text input
func (c *create) focusNext() {
	var nextInput input // if we are at the end, reset
	if int(c.focusedInput) == len(c.ti)-1 {
		nextInput = 0
	} else {
		nextInput = c.focusedInput + 1
	}

	c.ti[c.focusedInput].Blur() // unfocus current
	// focus next
	c.ti[nextInput].Focus()
	c.focusedInput = nextInput

}

// focusPrevious determines and focuses the previous text input
func (c *create) focusPrevious() {
	var nextInput input // if we are at the beginning, reset
	if c.focusedInput == 0 {
		nextInput = input(len(c.ti) - 1)
	} else {
		nextInput = c.focusedInput - 1
	}

	// unfocus current
	c.ti[c.focusedInput].Blur()
	// focus next (the prior ti)
	c.ti[nextInput].Focus()
	c.focusedInput = nextInput
}

// SetArgs parses the tokens against the local flagset and sets internal
// parameters. Returns false if the token set does not contain required flags or
// is invalid
func (c *create) SetArgs(_ *pflag.FlagSet, tokens []string) (invalid string, onStart []tea.Cmd, err error) {
	// parse the tokens agains the local flagset
	err = c.fs.Parse(tokens)
	if err != nil {
		return "", nil, err
	}

	// set action variable fields
	var val string
	if val, err = c.fs.GetString("name"); err != nil {
		return "", nil, err
	}
	val = strings.ToUpper(strings.TrimSpace(val))
	clilog.Writer.Debugf("Set name to %v", val)
	c.ti[name].SetValue(val)

	if val, err := c.fs.GetString("description"); err != nil {
		return "", nil, err
	} else if val != "" {
		clilog.Writer.Debugf("Set description to %v", val)
		c.ti[desc].SetValue(val)
	}

	if val, err := c.fs.GetString("expansion"); err != nil {
		return "", nil, err
	} else if val != "" {
		clilog.Writer.Debugf("Set expansion to %v", val)
		c.ti[value].SetValue(val)
	}

	return "", nil, nil
}

//#endregion

//#region help display

type helpKeyMap struct {
	Next key.Binding // tab
	Prev key.Binding // shift+tab
	Help key.Binding // F1
	Quit key.Binding // esc
}

func (k helpKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

func (k helpKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		// ! Bubbles transposes the bindings when displaying!
		{k.Help, k.Quit}, // first column
		{k.Next, k.Prev}, // second column
	}
}

//#endregion
