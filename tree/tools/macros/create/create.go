package create

import (
	"fmt"
	"gwcli/action"
	"gwcli/clilog"
	"gwcli/connection"
	"gwcli/treeutils"
	"unicode"

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

func createMacro(name, desc, value string) bool {
	// via the web gui, adding a macro requies a name and value (plus optional desc)
	macro := grav.SearchMacro{Name: name, Description: desc, Expansion: value}

	_, err := connection.Client.AddMacro(macro)
	if err != nil {
		clilog.Writer.Warnf("Failed to create Macro: %s", err.Error())
		return false
	}

	return true
}

//#region actor implementation

var promptWidth []int = []int{
	20, // name
	40, // desc
	60, // value
}

type input int

const (
	name input = iota
	desc
	value
)

type create struct {
	done bool

	focusedInput input

	ti []textinput.Model // name, desc, value
}

var Create action.Model = Initial()

func Initial() *create {
	c := &create{done: false}

	c.ti = make([]textinput.Model, 3)
	for i := 0; i < 3; i++ {
		c.ti[i] = textinput.New()
		c.ti[i].Width = promptWidth[i]
	}
	// the first ti (name) requires extra initialization (focus and validation)
	c.ti[0].Validate = func(s string) error {
		if err := grav.CheckMacroName(s); err != nil {
			return err
		}
		return nil
	}
	c.ti[0].Focus()
	c.focusedInput = name

	return c
}

func (c *create) Update(msg tea.Msg) tea.Cmd {

	if c.done {
		return nil
	}

	switch msg := msg.(type) { // check for meta inputs
	case tea.KeyMsg: // only KeyMsg could require special handling
		clilog.Writer.Debugf("key msg %v", msg.String())

		switch msg.Type {
		case tea.KeyEsc:
			c.done = true
			return nil

		case tea.KeyEnter:
			clilog.Writer.Debugf("Create.Update received enter %v", msg.String())
			if c.focusedInput == value { // if last input, attempt to create the macros
				c.done = createMacro(c.ti[name].Value(), c.ti[desc].Value(), c.ti[value].Value())
			} else {
				c.focusNext()
			}

		case tea.KeyTab:
			c.focusNext()

		case tea.KeyShiftTab:
			c.focusPrevious()

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

	clilog.Writer.Debugf("Passing updates to child tis %v", msg)

	var cmd tea.Cmd
	c.ti[c.focusedInput], cmd = c.ti[c.focusedInput].Update(msg)

	return cmd
}

func (c *create) View() string {
	return fmt.Sprintf("Name: %s\n"+
		"Desc: %s\n"+
		"Expansion: %s \n", c.ti[name].View(), c.ti[desc].View(), c.ti[value].View())
}

func (c *create) Done() bool {
	return c.done
}

/**
 * Reset clears the done flag and resets the model to its initial state,
 * dropping all data from each field.
 */
func (c *create) Reset() error {
	for i := range c.ti {
		c.ti[i].Reset()
	}

	c.ti[c.focusedInput].Blur()
	c.focusedInput = name
	c.ti[c.focusedInput].Focus()

	c.done = false
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

//#endregion
