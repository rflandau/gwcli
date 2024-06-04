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
	"gwcli/stylesheet"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

type navCmd = cobra.Command
type actionCmd = cobra.Command // actions have associated actors

type walkStatus int

const (
	invalidCommand walkStatus = iota
	foundNav
	foundAction
	foundBuiltin
	erroring
)

const (
	tiWidth int    = 40
	indent  string = "    "
)

// keys kill the program in Update no matter its other states
var killKeys = [...]tea.KeyType{tea.KeyCtrlC}

// special, global actions
// takes a reference to Mother and tokens [1:]
var builtins map[string](func(*Mother, []string) tea.Cmd)

func init() {
	// need init to avoid an initialization cycle
	builtins = map[string](func(*Mother, []string) tea.Cmd){
		"help": ContextHelp,
		"quit": quit,
		"exit": quit}
}

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
		args    []string     // arguments pass to action
	}
}

// internal new command to allow tests to pass in a renderer
func new(root *navCmd, pwd *navCmd, _ *lipgloss.Renderer) Mother {
	m := Mother{root: root, pwd: root, mode: prompting}
	if pwd != nil {
		m.pwd = pwd
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
	m.style.nav = stylesheet.NavStyle
	m.style.action = stylesheet.ActionStyle
	m.style.error = lipgloss.NewStyle().Foreground(lipgloss.Color("#CC444")).Bold(true)
	//}

	return m
}

/**
 * Generate a Mother instance to operate on the Cobra command tree.
 * If pwd is nil, Mother will start at root.
 */
func New(root *navCmd, pwd *navCmd) Mother {
	return new(root, pwd, nil)
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
		if keyMsg.Type == tea.KeyEsc {
			if m.active.model != nil {
				// kick out the child and return to normal processing
				clilog.Writer.Debugf("Escape. Mother reasserting...")
				m.UnsetAction()
				return m, nil
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
			m.UnsetAction()
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
	input := m.ti.Value()
	clilog.Writer.Debugf("Processing input '%s'\n", input)

	if m.ti.Err != nil {
		return nil
	}

	onComplete := make([]tea.Cmd, 1)
	onComplete[0] = tea.Println(m.promptString()) // save off prompt string to output as history
	m.ti.Reset()                                  // empty out the input

	// tokenize input
	given := strings.Split(strings.TrimSpace(input), " ")
	//m.ti.Validate(given) // TODO add navigation text validation

	/*var (
		endCmd     *cobra.Command
		status     walkStatus
		inputError string
	)*/
	//endCmd, status, onComplete, inputError = walk(m.pwd, given, onComplete)
	wr := walk(m.pwd, given, onComplete)

	if wr.errString != "" {
		return tea.Sequence(
			append(
				onComplete,
				tea.Println(m.style.error.Render(wr.errString)),
			)...)
	}
	// split on action or nav
	switch wr.status {
	case foundBuiltin:
		// if the built-in is not the first command, we don't care about it
		// so re-test only the first token
		if bi, ok := builtins[given[0]]; ok {
			onComplete = append(onComplete, bi(m, given[1:]))
			return tea.Sequence(onComplete...)
		}
	case foundNav:
		m.pwd = wr.endCommand
	case foundAction:
		m.mode = handoff

		// look up the subroutines to load
		m.active.model, _ = action.GetModel(wr.endCommand) // save add-on subroutines
		if m.active.model == nil {
			onComplete = append(
				onComplete,
				tea.Printf("Developer issue: Did not find actor associated to '%s'."+
					" Please submit a bug report.\n",
					wr.endCommand.Name()),
			)
		}
		// save relevant command and any extra tokens
		m.active.command = wr.endCommand
		m.active.args = wr.remainingTokens
	case invalidCommand:
		clilog.Writer.Errorf("walking input %v returned invalid", given)
	}

	return tea.Sequence(onComplete...)
}

//#region builtin functions

// Built-in, interactive help invocation
func ContextHelp(m *Mother, args []string) tea.Cmd {
	clilog.Writer.Debugf("Help with args(%d) '%v'", len(args), args)
	if len(args) == 0 {
		return TeaCmdContextHelp(m.pwd)
	}

	// walk the command tree
	// action or nav, print help about it
	// if invalid/no destination, print error
	wr := walk(m.pwd, args, make([]tea.Cmd, 1))

	if wr.errString != "" { // erroneous input
		return tea.Println(m.style.error.Render(wr.errString))
	}
	switch wr.status {
	case foundNav, foundAction:
		return TeaCmdContextHelp(wr.endCommand)
	case foundBuiltin:
		if _, ok := builtins[args[0]]; ok {
			// TODO fill in help information for each built-in command
			return tea.Printf("help for %v", args[0])
		}

	}

	clilog.Writer.Debugf("Doing nothing (%#v)", wr)

	return nil
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

/**
 * UnsetAction resets the current active command/action, clears actives, and
 * returns control to Mother.
 */
func (m *Mother) UnsetAction() {
	if m.active.model == nil || m.active.command == nil {
		panic("nil actives")
	}
	m.active.model.Reset()
	m.mode = prompting
	m.active.model = nil
	m.active.command = nil
}

//#endregion

//#region static helper functions

//#region tree navigation

type walkResult struct {
	endCommand     *cobra.Command // the relevent command walk completed on
	status         walkStatus     // ending state
	onCompleteCmds []tea.Cmd      // ordered list of commands to pass to the bubble tea driver
	errString      string

	// builtin function information, if relevant (else Zero vals)
	builtinStr  string
	builtinFunc func(*Mother, []string) tea.Cmd

	// contains args for actions
	remainingTokens []string // any tokens remaining for later processing by walk caller
}

// Recursively walk the tokens of the exploded user input until we run out or
// find a valid destination.
// Returns the relevant command (ending Nav destination or action to invoke),
// the type of the command (action, nav, invalid), a list of commands to pass to
// tea, and an error (if one occurred).
func walk(dir *cobra.Command, tokens []string, onCompleteCmds []tea.Cmd) walkResult {
	if len(tokens) == 0 {
		// only move if the final command was a nav
		return walkResult{
			endCommand:     dir,
			status:         foundNav,
			onCompleteCmds: onCompleteCmds,
		}
	}

	curToken := strings.TrimSpace(tokens[0])

	if bif, ok := builtins[curToken]; ok { // check for built-in command
		return walkResult{
			endCommand:      nil,
			status:          foundBuiltin,
			onCompleteCmds:  onCompleteCmds,
			builtinStr:      curToken,
			builtinFunc:     bif,
			remainingTokens: tokens[1:],
		}
	}

	if curToken == ".." { // navigate upward
		dir = up(dir)
		return walk(dir, tokens[1:], onCompleteCmds)
	}

	// test for a local command
	var invocation *cobra.Command = nil
	for _, c := range dir.Commands() {

		if c.Name() == curToken { // check name
			invocation = c
			clilog.Writer.Debugf("Match, invoking %s", invocation.Name())
		} else { // check aliases
			for _, alias := range c.Aliases {
				if alias == curToken {
					invocation = c
					clilog.Writer.Debugf("Alias match, invoking %s", invocation.Name())
					break
				}
			}
		}
		if invocation != nil {
			break
		}
	}

	// check if we found a match
	if invocation == nil {
		// user request unhandlable
		return walkResult{
			endCommand:      nil,
			status:          invalidCommand,
			onCompleteCmds:  onCompleteCmds,
			errString:       fmt.Sprintf("unknown command '%s'. Press F1 or type 'help' for relevant commands.", curToken),
			remainingTokens: tokens[1:],
		}
	}

	// split on action or nav
	if action.Is(invocation) {
		return walkResult{
			endCommand:      invocation,
			status:          foundAction,
			onCompleteCmds:  onCompleteCmds,
			remainingTokens: tokens[1:],
		}
	} else { // nav
		// navigate given path
		dir = invocation
		return walk(dir, tokens[1:], onCompleteCmds)
	}
}

//#endregion

// Return the parent directory to the given command
func up(dir *cobra.Command) *cobra.Command {
	if dir.Parent() == nil { // if we are at root, do nothing
		return dir
	}
	// otherwise, step upward
	clilog.Writer.Debugf("Up: %v -> %v", dir.Name(), dir.Parent().Name())
	return dir.Parent()
}

/* Returns a tea.Println Cmd containing the path to the pwd */
func TeaCmdPath(c *cobra.Command) tea.Cmd {
	return tea.Println(c.CommandPath())
}

/* Quit the program */
func quit(*Mother, []string) tea.Cmd {
	return tea.Sequence(tea.Println("Bye"), tea.Quit)
}

/**
 * Returns a tea.Println Cmd containing the context help for the given command.
 * Structure:
 * <nav> - <desc>
 *     <childnav> <childaction> <childnav>
 * <nav> - <desc>
 *     <childaction>
 * <action> - <desc>
 *
 */
func TeaCmdContextHelp(c *cobra.Command) tea.Cmd {
	// generate a list of all available Navs and Actions with their associated shorts
	var s strings.Builder

	if action.Is(c) {
		s.WriteString(c.UsageString())
	} else {
		children := c.Commands()
		for _, child := range children {
			// handle special commands
			if child.Name() == "help" || child.Name() == "completion" {
				continue
			}
			var name string
			var subchildren strings.Builder // children of this child
			if action.Is(child) {
				name = stylesheet.ActionStyle.Render(child.Name())
			} else {
				name = stylesheet.NavStyle.Render(child.Name())
				// build and color subchildren
				for _, sc := range child.Commands() {
					_, err := subchildren.WriteString(stylesheet.ColorCommandName(sc) + " ")
					if err != nil {
						panic(err)
					}
				}

			}
			// generate the output
			trimmedSubChildren := strings.TrimSpace(subchildren.String())
			s.WriteString(fmt.Sprintf("%s%s - %s\n", indent, name, child.Short))
			if trimmedSubChildren != "" {
				s.WriteString(indent + indent + trimmedSubChildren + "\n")
			}
		}
	}

	// TODO store the string within mother somewhere so we can lazy-compile all strings
	// chomp last newline and return
	return tea.Println(strings.TrimSuffix(s.String(), "\n"))
}

func CommandPath(m *Mother) string {
	return m.style.nav.Render(m.pwd.CommandPath())
}

//#endregion
