/**
 * Mother manages the interactive body of gwcli.
 * It is the local implementation of tea.Model and drives interactive tree
 * navigation as well as managing of child processing (Actions).
 */

package mother

import (
	"fmt"
	"gwcli/actor"
	"gwcli/connection"
	"gwcli/treeutils"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gravwell/gravwell/v3/ingest/log"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

type nav = cobra.Command
type action = cobra.Command // actions have associated actors

const (
	tiWidth int    = 40
	indent  string = "    "
)

// keys kill the program in Update no matter its other states
var killKeys = [...]tea.KeyType{tea.KeyCtrlC, tea.KeyEsc}

var builtins = map[string](func(*Mother) tea.Cmd){
	"..":   navParent,
	"help": ContextHelp,
	"quit": quit,
	"exit": quit}

var actors = map[string]actor.Model{}

/* tea.Model implementation, carrying all data required for interactive use */
type Mother struct {
	mode mode

	// tree references
	root *nav
	pwd  *nav

	style struct {
		nav    lipgloss.Style
		action lipgloss.Style
		error  lipgloss.Style
	}

	ti textinput.Model

	log    *log.Logger
	active struct {
		command *action     // command user called
		actor   actor.Model // Elm Arch associated to command
	}
}

// internal new command to allow tests to pass in a renderer
func new(root *nav, _ *lipgloss.Renderer) Mother {
	var err error
	m := Mother{root: root, pwd: root, mode: prompting}

	// logger
	m.log, err = log.NewFile("gwcli.log") // TODO allow external log level edits and output redirection
	if err != nil {
		panic(err)
	}
	if err = m.log.SetLevel(log.DEBUG); err != nil { // TODO make the logger terse by default
		panic(err)
	}

	// text input
	m.ti = textinput.New()
	m.ti.Placeholder = "help"
	m.ti.Focus()
	m.ti.Width = tiWidth

	// stylesheet
	/*if r != nil { // given renderer
		// TODO
	} else { */ // auto-selected renderer
	m.style.nav = treeutils.NavStyle
	m.style.action = treeutils.ActionStyle
	m.style.error = lipgloss.NewStyle().Foreground(lipgloss.Color("#CC444")).Bold(true)
	//}

	return m
}

/* Generate a Mother instance to operate on the Cobra command tree */
func New(root *nav) Mother {
	return new(root, nil)
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
				return m, tea.Batch(tea.Quit, connection.End, tea.Println("Bye"))
			}
		}
	}

	// a child is running
	if m.mode == handoff {
		if m.active.actor == nil || m.active.command == nil { // sanity check
			panic(fmt.Sprintf(
				"Mother is in handoff mode but has inconsistent actives %#v",
				m.active))
		}
		// test for child state
		if !m.active.actor.Done() { // child still processing
			m.log.Debugf("Handing off Update to %s\n", m.active.command.Name())
			return m, m.active.actor.Update(&msg)
		} else {
			// child has finished processing, regain control and return to normal processing
			m.log.Debugf("Child %s done. Mother reasserting...", m.active.command.Name())
			go m.active.actor.Reset()
			m.mode = prompting
			m.active.actor = nil
			m.active.command = nil
		}
	}

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
			cmd := processInput(&m)
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
	s.WriteString(CommandPath(&m) + m.ti.View()) // prompt
	// TODO currently superfluous
	if m.ti.Err != nil {
		// write out previous error and clear it
		s.WriteString("\n")
		s.WriteString(m.style.error.Render(m.ti.Err.Error()))
		m.ti.Err = nil
		// this will be cleared from view automagically on next key input
	}
	return s.String()
}

//#endregion

/**
 * processInput consumes and clears the text on the prompt, determines what
 * action to take, modifies the model accordingly, and outputs the state of the
 * prompt as a newline.
 * ! Be sure each path that clears the prompt also outputs it via tea.Println
 */
func processInput(m *Mother) tea.Cmd {
	var given string = m.ti.Value()
	m.log.Debugf("Processing input '%s'\n", given)
	//m.ti.Validate(given) // TODO add navigation text validation
	if m.ti.Err != nil {
		return nil
	}

	priorL := m.promptString() // save off prompt string to output as history
	m.ti.Reset()               // empty out the input

	// check for a builtin command
	builtinFunc, ok := builtins[given]
	if ok {
		return tea.Sequence(tea.Println(priorL), builtinFunc(m))
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
	}

	// check if we found a match
	if invocation == nil {
		// user request unhandlable

		return tea.Sequence(
			tea.Println(priorL),
			tea.Println(m.style.error.Render(fmt.Sprintf("unknown command '%s'. Press F1 or type 'help' for relevant commands.", given))),
		)
	}

	// split on action or nav
	if isAction(invocation) { // hand off control to child
		m.mode = handoff

		// look up the subroutines to load
		var ok bool
		m.active.actor, ok = actors[invocation.CommandPath()] // save add-on subroutines
		if !ok {
			m.active.actor = nil
			return tea.Sequence(
				tea.Println(priorL),
				tea.Println(m.style.error.Render(fmt.Sprintf("Developer issue: no actor associated to action %s. Please submit a bug report.", given))),
			)
		}
		m.active.command = invocation // save relevant command
		return tea.Println(priorL)
	} else { // nav
		// navigate to child
		m.pwd = invocation
		return tea.Println(priorL)
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
	return nil
}

func ContextHelp(m *Mother) tea.Cmd {
	return TeaCmdContextHelp(m.pwd)
}

//#endregion

//#region helper functions

/* Returns a composition resembling the full prompt. */
func (m *Mother) promptString() string {
	return fmt.Sprintf("%s> %s", CommandPath(m), m.ti.Value())
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
	// generate a list of all available Navs and Actions with their associated shorts
	var s strings.Builder

	children := c.Commands()
	for _, child := range children {
		// handle special commands
		if child.Name() == "help" || child.Name() == "completion" {
			continue
		}
		var name string
		if isAction(child) {
			name = treeutils.ActionStyle.Render(child.Name())
		} else {
			name = treeutils.NavStyle.Render(child.Name())
		}
		s.WriteString(fmt.Sprintf("%s%s - %s\n", indent, name, child.Short))
	}

	/* Old form using Cobra's standard help template
	// redirect output so we can pass it to bubble tea
	var s strings.Builder
	c.SetOut(&s) // TODO do this on mother's invocation?
	c.Usage()    // TODO do not print flags?
	*/

	// TODO store the string within mother somewhere so we can lazy-compile all strings
	// chomp last newline and return
	return tea.Println(strings.TrimSuffix(s.String(), "\n"))
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

func CommandPath(m *Mother) string {
	return m.style.nav.Render(m.pwd.CommandPath())
}

//#endregion
