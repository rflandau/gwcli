package mother

import (
	"gwcli/treeutils"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gravwell/gravwell/v3/ingest/log"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

type nav = cobra.Command

const (
	tiWidth int = 40
)

// keys kill the program in Update no matter its other states
var killKeys = [...]tea.KeyType{tea.KeyCtrlC, tea.KeyEsc}

var builtins = map[string](func(*Mother) tea.Cmd){
	"..":   navParent,
	"help": ContextHelp,
	"quit": quit,
	"exit": quit}

/* tea.Model implementation, carrying all data required for interactive use */
type Mother struct {
	mode

	// tree references
	root *nav
	pwd  *nav

	style struct {
		nav    lipgloss.Style
		action lipgloss.Style
	}

	ti textinput.Model

	log *log.Logger
}

/* Generate a Mother instance to operate on the Cobra command tree */
func New(root *nav) Mother {
	var err error
	m := Mother{root: root, pwd: root, mode: prompting}

	// logger
	m.log, err = log.NewFile("gwcli.log") // TODO allow external log level edits and output redirection
	if err != nil {
		panic(err)
	}
	m.log.SetLevel(log.DEBUG)

	// text input
	m.ti = textinput.New()
	m.ti.Placeholder = "help"
	m.ti.Focus()
	m.ti.Width = tiWidth

	// stylesheet
	m.style.nav = treeutils.NavStyle
	m.style.action = treeutils.ActionStyle

	return m
}

//#region tea.Model implementation

var _ tea.Model = Mother{}

func (m Mother) Init() tea.Cmd {
	return textinput.Blink
}

/** Inputs are handled in two places:
 * Persistent keystrokes (ex: F1, CTRL+C) are handled here (as kill keys).
 * Input commands (ex: 'help', 'quit', <command>) are handled in processInput()
 */
func (m Mother) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	// always handle kill keys
	keyMsg, isKeyMsg := msg.(tea.KeyMsg)
	if isKeyMsg {
		for _, v := range killKeys {
			if keyMsg.Type == v {
				m.mode = quitting
				return m, tea.Batch(tea.Quit, tea.Println("Bye"))
			}
		}
	}

	// manage child action if in handoff mode
	/*if m.mode == handoff {
		if m.activeCommand == "" { // sanity check
			panic(fmt.Sprintf("active command (%s) and mode (%s) are inconsistent", m.activeCommand, m.mode.String()))
		}

		if m.actions[m.activeCommand].Done() { // return to normal processing
			m.log.Println("Returning from command...")
			// ensure we are in an active command
			if m.activeCommand == "" {
				panic("return mode but no active command")
			}
			m.activeCommand = ""
			m.mode = prompting
		} else { // hand control to child action
			m.log.Printf("Handing off Update control to active command %s\n", m.activeCommand)
			return m, m.actions[m.activeCommand].Update(&msg)
		}
	} */

	// normal handling
	switch msg := msg.(type) {
	/*case message.Err:
	m.err = msg
	return m, tea.Sequence(tea.Println("Bye"), tea.Quit) */
	case tea.KeyMsg:
		// NOTE kill keys are handled above
		if msg.Type == tea.KeyF1 { // help
			return m, ContextHelp(&m)
		}
		if msg.Type == tea.KeyEnter { // submit
			//m.log.Printf("User hit enter, parsing data '%v'\n", m.ti.Value())

			// on enter, clear any error text, process the data in the text
			// input, and manipulate model accordingly

			//m.inputErr = nil
			m.log.Debugf("Mother's old pwd is %s", m.pwd.Name())
			cmd := processInput(&m)
			/*if m.inputErr != nil {
				m.log.Printf("%v\n", m.inputErr)
			}*/
			m.log.Debugf("Mother's new pwd is %s", m.pwd.Name())
			return m, cmd
		}
	}

	var cmd tea.Cmd
	m.ti, cmd = m.ti.Update(msg)

	return m, cmd
}

func (m Mother) View() string {
	// allow child command to retain control if it exists
	/*
		if m.activeCommand != "" {
			return m.actions[m.activeCommand].View()
		}
	*/

	// if there was a fatal error, print it out and return
	/*
		if m.err != nil {
			return fmt.Sprintf("\nErr: %v\n\n", m.err)
		}
	*/

	s := strings.Builder{}
	s.WriteString(m.pwd.CommandPath() + m.ti.View()) // prompt
	if m.ti.Err != nil {
		// TODO
		// write out previous error and clear it
		s.WriteString("\n")
		//s.WriteString(m.ss.ErrorText.Render(m.ti.Err.Error()))
		m.ti.Err = nil
		// this will be cleared from view automagically on next key input
	}
	return s.String()
}

//#endregion

/**
 * processInput consumes and clears the text on the prompt, determining what action to take and modifying the model accordingly.
 */
func processInput(m *Mother) tea.Cmd {
	var given string = m.ti.Value()
	m.log.Debugf("Processing input '%s'\n", given)
	//m.ti.Validate(given) // TODO add navigation text validation
	if m.ti.Err != nil {
		return nil
	}
	m.ti.Reset() // empty out the input
	// check for a builtin command
	builtinFunc, ok := builtins[given]
	if ok {
		return builtinFunc(m)
	}
	// if we do not find a built in, test for a valid invocation
	var invocation *cobra.Command = nil
	for _, c := range m.pwd.Commands() {
		// TODO incorporate aliases
		m.log.Debugf("Given '%s' =?= child '%s'", given, c.Name())

		if c.Name() == given { // match
			invocation = c
			m.log.Debugf("Match, invoking %s", invocation.Name())
			break
		}
		//m.log.Printf("\n", given, c.Name()) // DEBUG
	}

	// check if we found a match
	if invocation == nil {
		// user request unhandlable
		//m.inputErr = fmt.Errorf("%s has no child '%s'", m.PWD.Name(), given)
		return nil
	}

	// split on action or nav
	if isAction(invocation) {
		// hand off control to child
		//m.log.Printf("Found local command %v\n", invocation.Name())
		//m.mode = handoff
		// TODO each time a command is called, it should be instantiated fresh so old data does not garble the new call
		//m.activeCommand = invocation.CommandPath()
		return nil
	} else { // nav
		// navigate to child
		m.log.Debug("Replacing PWD")
		m.pwd = invocation
		return nil
	}
}

//#region builtin functions

/* Using the current menu, navigate up one level */
func navParent(m *Mother) tea.Cmd {
	if m.pwd == m.root { // if we are at root, do nothing
		return nil
	}
	// otherwise, step upward
	m.pwd = m.pwd.Parent()
	return TeaCmdPath(m.pwd)
}

func ContextHelp(m *Mother) tea.Cmd {
	return TeaCmdContextHelp(m.pwd)
}

//#endregion

//#region static helper functions

/* Returns a tea.Println Cmd containing the path to the pwd */
func TeaCmdPath(c *cobra.Command) tea.Cmd {
	return tea.Println(c.CommandPath())
}

/* Quit the program */
func quit(*Mother) tea.Cmd {
	return tea.Sequence(tea.Println("Bye"), tea.Quit)
}

/* Returns a tea.Println Cmd containing the context help for the given command */
func TeaCmdContextHelp(c *cobra.Command) tea.Cmd {
	// redirect output so we can pass it to bubble tea
	var s strings.Builder
	c.SetOut(&s) // TODO do this on mother's invocation?
	c.Usage()    // TODO do not print flags?

	return tea.Println(s.String())
}

/**
 * Given a cobra.Command, returns whether it is an Action (and thus can supplant
 * Mother's Elm cycle) or a Nav.
 */
func isAction(cmd *cobra.Command) bool {
	if cmd == nil { // sanity check
		panic("cmd cannot be nil!")
	}
	// does not `return cmd.GroupID == treeutils.ActionID` to facilitate sanity check
	switch cmd.GroupID {
	case treeutils.ActionID:
		return true
	case treeutils.NavID:
		return false
	default: // sanity check
		panic("cmd '" + cmd.Name() + "' is neither a nav nor an action!")
	}
}

//#endregion
