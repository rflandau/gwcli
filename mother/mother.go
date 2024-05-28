/**
 * Mother manages the interactive body of gwcli.
 * It is the local implementation of tea.Model and drives interactive tree
 * navigation as well as managing of child processing (Actions).
 */

package mother

import (
	"fmt"
	"gwcli/action"
	"gwcli/clilog"
	"gwcli/connection"
	"gwcli/treeutils"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

type navCmd = cobra.Command
type actionCmd = cobra.Command // actions have associated actors

const (
	tiWidth int    = 40
	indent  string = "    "
)

// keys kill the program in Update no matter its other states
var killKeys = [...]tea.KeyType{tea.KeyCtrlC}

var builtins = map[string](func(*Mother) tea.Cmd){
	"..":   navParent,
	"help": ContextHelp,
	"quit": quit,
	"exit": quit}

/* tea.Model implementation, carrying all data required for interactive use */
type Mother struct {
	mode mode

	// tree references
	root *navCmd
	pwd  *navCmd

	style struct {
		nav    lipgloss.Style
		action lipgloss.Style
		error  lipgloss.Style
	}

	ti textinput.Model

	active struct {
		command *actionCmd   // command user called
		model   action.Model // Elm Arch associated to command
	}
}

// internal new command to allow tests to pass in a renderer
func new(root *navCmd, _ *lipgloss.Renderer) Mother {
	m := Mother{root: root, pwd: root, mode: prompting}

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
func New(root *navCmd) Mother {
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
			// TODO if we receive a kill key in a child command, kill just the child
			if keyMsg.Type == v {
				m.mode = quitting
				return m, tea.Batch(tea.Quit, connection.End, tea.Println("Bye"))
			}
		}
	}

	// a child is running
	if m.mode == handoff {
		if m.active.model == nil || m.active.command == nil { // sanity check
			panic(fmt.Sprintf(
				"Mother is in handoff mode but has inconsistent actives %#v",
				m.active))
		}
		// test for child state
		if !m.active.model.Done() { // child still processing
			clilog.Writer.Debugf("Handing off Update to %s\n", m.active.command.Name())
			return m, m.active.model.Update(msg)
		} else {
			// child has finished processing, regain control and return to normal processing
			clilog.Writer.Debugf("Child %s done. Mother reasserting...", m.active.command.Name())
			m.active.model.Reset()
			m.mode = prompting
			m.active.model = nil
			m.active.command = nil
		}
	}

	// normal handling
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// NOTE kill keys are handled above
		if msg.Type == tea.KeyF1 { // help
			return m, m.f1Help()
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
	if m.active.model != nil {
		return m.active.model.View()
	}

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
	// TODO trim inputs
	var given string = strings.TrimSpace(m.ti.Value())
	clilog.Writer.Debugf("Processing input '%s'\n", given)
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
		clilog.Writer.Debugf("Given '%s' =?= child '%s'", given, c.Name())

		if c.Name() == given { // match
			invocation = c
			clilog.Writer.Debugf("Match, invoking %s", invocation.Name())
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
	if action.Is(invocation) { // hand off control to child
		m.mode = handoff

		// look up the subroutines to load
		m.active.model, _ = action.GetModel(invocation) // save add-on subroutines
		if m.active.model == nil {
			return tea.Sequence(
				tea.Println(priorL),
				tea.Println(m.style.error.Render(fmt.Sprintf("Developer issue: Did not find actor associated to '%s'. Please submit a bug report.", given))))
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

/**
 * f1Help displays context help relevant to the current state of the model.
 * It determines if F1 contextual help should be relevant to the pwd or a
 * command currently on the prompt
 */
func (m *Mother) f1Help() tea.Cmd {
	// figure out the current state of the prompt
	var prompt string = strings.TrimSpace(m.ti.Value())
	if prompt == "" {
		// show help for current directory
		return tea.Sequence(tea.Println(m.promptString()), TeaCmdContextHelp(m.pwd))
	}
	// check if prompt has relevant info
	var children []*cobra.Command = m.pwd.Commands()
	clilog.Writer.Debugf("Context Help || prompt: '%s'|pwd:'%s'|children:'%v'", prompt, m.pwd.Name(), children)
	for _, child := range children {
		if child.Name() == prompt {
			return tea.Sequence(tea.Println(m.promptString()), TeaCmdContextHelp(child))
		}
	}
	// no matches
	clilog.Writer.Debug("no matching child found")
	return tea.Sequence(tea.Println(m.promptString()), TeaCmdContextHelp(m.pwd))
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
		if action.Is(child) {
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

func CommandPath(m *Mother) string {
	return m.style.nav.Render(m.pwd.CommandPath())
}

//#endregion
