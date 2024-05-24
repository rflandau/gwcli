package create

import (
	"fmt"
	"gwcli/action"
	"gwcli/treeutils"

	grav "github.com/gravwell/gravwell/v3/client/types"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func GenerateAction() action.Pair {
	return treeutils.GenerateAction("create", "create a new macro", "", []string{}, run, Create)
}

func run(_ *cobra.Command, _ []string) {
	fmt.Println("create macro")
}

func createMacro() {
	// via the web gui, adding a macro requies a name and value (plus optional desc)
}

//#region actor implementation

const (
	namePromptWidth  int = 20
	descPromptWidth  int = 40
	valuePromptWidth int = 60
)

type input int

const (
	name input = iota
	desc
	value
)

type create struct {
	done bool

	focusedInput input

	// text inputs
	tis struct {
		name  textinput.Model
		desc  textinput.Model
		value textinput.Model
	}
}

var Create action.Model = Initial()

func Initial() *create {
	c := &create{done: false}

	// initialize all text inputs
	c.tis.name = textinput.New()
	c.tis.name.Validate = func(s string) error {
		if err := grav.CheckMacroName(s); err != nil {
			return err
		}
		return nil
	}
	c.tis.name.Focus()
	c.focusedInput = name
	c.tis.name.Width = namePromptWidth

	c.tis.desc = textinput.New()
	c.tis.desc.Width = descPromptWidth

	c.tis.value = textinput.New()
	c.tis.desc.Width = valuePromptWidth

	return c
}

func (c *create) Update(msg tea.Msg) tea.Cmd {

	if msg, ok := msg.(tea.KeyMsg); ok { // check for meta inputs
		// only KeyMsg could require special handling
		switch msg.Type {
		case tea.KeyEnter:
			// if last input, attempt to create the macros
			if c.focusedInput == value {
				// TODO
			} else {
				c.focusNext()
			}
			// TODO handle tab and shift tab navigation

		}
	}

	c.done = true
	return nil
}

func (c *create) View() string {
	// TODO
	return ""
}

func (c *create) Done() bool {
	return c.done
}

/**
 * Reset clears the done flag and resets the model to its initial state,
 * dropping all data from each field.
 */
func (c *create) Reset() error {
	c.tis.name = textinput.New()

	c.done = false
	// TODO
	return nil
}

func (c *create) focusNext() {
	c.focusedInput += 1
	// TODO focus next ti
}

//#endregion
