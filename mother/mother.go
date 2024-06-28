/**
 * Mother is the heart and brain of the interactive functionality of gwcli.
 * It is the top-level implementation of tea.Model and drives interactive tree
 * navigation as well as managing of child processing (Actions).
 * Almost all interactivity flows through Mother, even when a child is in
 * control (aka: Mother is in handoff mode); Mother's Update() and View() are
 * still called every cycle, but control rapidly passes to the child's Update()
 * and View().
 * Mother also manages the top-level prompt.
 */

package mother

import (
	"fmt"
	"gwcli/action"
	"gwcli/clilog"
	"gwcli/connection"
	"gwcli/stylesheet"
	"gwcli/stylesheet/colorizer"
	"gwcli/utilities/killer"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gravwell/gravwell/v3/ingest/log"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type navCmd = cobra.Command
type actionCmd = cobra.Command // actions have associated actors via the action map

const (
	indent string = "    "
)

func init() {
	initBuiltins() // need init to avoid an initialization cycle
}

// Mother, a struct satisfying the tea.Model interface and containing information required for
// cobra.Command tree traversal.
// Facillitates interactive use of gwcli.
type Mother struct {
	mode mode

	// tree references
	root *navCmd
	pwd  *navCmd

	ti textinput.Model

	active struct {
		command *actionCmd   // command user called
		model   action.Model // Elm Arch associated to command
	}

	history *history
}

// internal new command to allow tests to pass in a renderer
func new(root *navCmd, pwd *navCmd, _ *lipgloss.Renderer) Mother {
	m := Mother{root: root, pwd: root, mode: prompting}
	if pwd != nil {
		m.pwd = pwd
	}

	// disable completions command when mother is spun up
	if c, _, err := root.Find([]string{"completion"}); err != nil {
		clilog.Writer.Warnf("failed to disable 'completion' command: %v", err)
	} else if c != nil {
		root.RemoveCommand(c)
	}

	// text input
	m.ti = textinput.New()
	m.ti.Placeholder = "help"
	m.ti.Prompt = stylesheet.TIPromptPrefix
	m.ti.Focus()
	m.ti.Width = stylesheet.TIWidth
	// add ctrl+left/right to the word traversal keys
	m.ti.KeyMap.WordForward.SetKeys("ctrl+right", "alt+right", "alt+f")
	m.ti.KeyMap.WordBackward.SetKeys("ctrl+left", "alt+left", "alt+b")

	m.ti.ShowSuggestions = true
	m.updateSuggestions()

	m.history = newHistory()

	return m
}

// Generate a Mother instance to operate on the Cobra command tree.
// If pwd is nil, Mother will start at root.
func New(root *navCmd, pwd *navCmd) Mother {
	return new(root, pwd, nil)
}

//#region tea.Model implementation

var _ tea.Model = Mother{}

func (m Mother) Init() tea.Cmd {
	return textinput.Blink
}

// Mother's Update is always the entrypoint for BubbleTea to drive.
// It checks for kill keys (to disallow a runaway/ill-designed child), then either passes off
// control (if in handoff mode) or handles the input itself (if in prompt mode).
func (m Mother) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch killer.CheckKillKeys(msg) { // handle kill keys above all else
	case killer.Global:
		// if in handoff mode, just kill the child
		if m.mode == handoff {
			m.unsetAction()
			// if we are killing from mother, we must manually exit alt screen
			// (harmless if not in use)
			return m, tea.Batch(tea.ExitAltScreen, textinput.Blink)
		}
		connection.End()
		return m, tea.Batch(tea.Println("Bye"), tea.Quit)
	case killer.Child: // ineffectual if not in handoff mode
		m.unsetAction()
		return m, tea.Batch(tea.ExitAltScreen, textinput.Blink)
	}

	if m.mode == handoff { // a child is running
		activeChildSanityCheck(m)
		// test for child state
		if !m.active.model.Done() { // child still processing
			return m, m.active.model.Update(msg)
		} else {
			// child has finished processing, regain control and return to normal processing
			m.unsetAction()
			return m, textinput.Blink
		}
	}

	// normal handling
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// NOTE kill keys are handled above
		if msg.Type == tea.KeyF1 { // help
			return m, contextHelp(&m, strings.Split(strings.TrimSpace(m.ti.Value()), " "))
		}
		if msg.Type == tea.KeyUp { // history
			m.ti.SetValue(m.history.getOlderRecord())
			// update cursor position
			m.ti.CursorEnd()
			return m, nil
		}
		if msg.Type == tea.KeyDown { // history
			m.ti.SetValue(m.history.getNewerRecord())
			// update cursor position
			m.ti.CursorEnd()
			return m, nil
		}
		if msg.Type == tea.KeyEnter { // submit
			m.history.unsetFetch()
			cmd := processInput(&m)
			return m, cmd
		}
	}

	var cmd tea.Cmd
	m.ti, cmd = m.ti.Update(msg)

	return m, cmd
}

// helper function for m.Update.
// Validates that mother's active states have not become corrupted by a bug elsewhere in the code.
// Panics if it detects an error
func activeChildSanityCheck(m Mother) {
	if m.active.model == nil || m.active.command == nil {
		clilog.Writer.Warnf(
			"Mother is in handoff mode but has inconsistent actives %#v",
			m.active)
		if m.active.command == nil {
			clilog.Writer.Warnf("nil command, unable to recover. Dying...")
			panic("inconsistent handoff mode. Please submit a bug report.")
		}
		// m.active.model == nil, !m.active.command
		var err error
		m.active.model, err = action.GetModel(m.active.command)
		if err != nil {
			clilog.Writer.Errorf("failed to recover model from command: %v", err)
			panic("inconsistent handoff mode. Please submit a bug report. ")
		}
	}
}

func (m Mother) View() string {
	// allow child command to retain control if it exists
	if m.active.model != nil {
		return m.active.model.View()
	}
	return fmt.Sprintf("%s%v\n",
		CommandPath(&m), m.ti.View())
}

//#endregion

// processInput consumes and clears the text on the prompt, determines what action to take, modifies
// the model accordingly, and outputs the state of the prompt as a newline.
// ! Be sure each path that clears the prompt also outputs it via tea.Println
func processInput(m *Mother) tea.Cmd {
	// sanity check error state of the ti
	if m.ti.Err != nil {
		clilog.Writer.Warnf("text input has a reported error: %v", m.ti.Err)
		m.ti.Err = nil
	}

	onComplete := make([]tea.Cmd, 1) // tea.Cmds to execute on completion
	var input string
	var err error
	onComplete[0], input, err = m.pushToHistory()
	if err != nil {
		clilog.Writer.Warnf("pushToHistory returned %v", err)
		return nil
	}

	// tokenize input
	given := strings.Split(strings.TrimSpace(input), " ")

	wr := walk(m.pwd, given, onComplete)
	if wr.errString != "" {
		return tea.Sequence(
			append(
				onComplete,
				tea.Println(stylesheet.ErrStyle.Render(wr.errString)),
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
		m.pwd = wr.endCommand // move mother to target directory
		// update her suggestions
		m.updateSuggestions()
	case foundAction:
		m.mode = handoff

		// look up the subroutines to load
		m.active.model, _ = action.GetModel(wr.endCommand) // save add-on subroutines
		if m.active.model == nil {
			// undo handoff mode
			m.mode = prompting

			return tea.Sequence(append(
				onComplete,
				tea.Printf("Developer issue: Did not find actor associated to '%s'."+
					" Please submit a bug report.\n",
					wr.endCommand.Name()))...)
		}
		m.active.command = wr.endCommand

		var fStr strings.Builder

		// don't bother visiting if it won't be printed
		if clilog.Writer.GetLevel() == log.DEBUG {
			m.active.command.InheritedFlags().Visit(func(f *pflag.Flag) {
				fStr.WriteString(
					fmt.Sprintf("%s - %s", f.Name, f.Value),
				)
			})
		}

		clilog.Writer.Debugf("Passing args (%v) and inherited flags (%#v) into %s\n",
			wr.remainingTokens,
			fStr.String(),
			m.active.command.Name())

		// NOTE: the inherited flags here may have a combination of parsed and !parsed flags
		// persistent commands defined below root may not be parsed
		if invalid, cmds, err := m.active.model.SetArgs(m.active.command.InheritedFlags(), wr.remainingTokens); err != nil {
			m.unsetAction()

			errString := fmt.Sprintf("Failed to set args %v: %v", wr.remainingTokens, err)
			clilog.Writer.Errorf("%v\nactive model %v\nactive command%v", errString, m.active.model, wr.endCommand)

			return tea.Sequence(append(
				onComplete,
				tea.Println(errString))...)
		} else if invalid != "" {
			return tea.Sequence(append(
				onComplete,
				tea.Println("invalid arguments. See help for invocation requirements"))...)
		} else if cmds != nil {
			onComplete = append(onComplete, cmds...)
		}

		clilog.Writer.Debugf("Handing off control to %s", m.active.command.Name())

	case invalidCommand:
		clilog.Writer.Errorf("walking input %v returned invalid", given)
	}

	// do not include pushToHistory in the sequence to ensure it doesn't get delayed due to
	// sequencing
	return tea.Batch(onComplete[0], tea.Sequence(onComplete[1:]...))
}

// pushToHistory generates and stores historical record of the prompt (as a
// Println and in the history array) and then clears the prompt, returning
// cleaned, usable user input
func (m *Mother) pushToHistory() (println tea.Cmd, userIn string, err error) {
	userIn = m.ti.Value()
	if m.ti.Err != nil {
		return nil, userIn, m.ti.Err
	}
	p := m.promptString()

	m.history.insert(userIn)           // add prompt string to history
	m.ti.Reset()                       // empty out the input
	return tea.Println(p), userIn, nil // print prompt
}

// Returns a composition resembling the full prompt.
func (m *Mother) promptString() string {
	return fmt.Sprintf("%s> %s", CommandPath(m), m.ti.Value())
}

// Call *after* moving to update the current command suggestions
func (m *Mother) updateSuggestions() {
	var suggest []string
	children := m.pwd.Commands()
	suggest = make([]string, len(children)+len(builtins))
	// add builtins
	var i int = 0
	for k := range builtins {
		suggest[i] = k
		i++
	}

	// add current sub commands
	for _, c := range children {
		// disable cobra-special commands
		if c.Name() == "help" || c.Name() == "completions" {
			continue
		}
		suggest[i] = c.Name()
		i++
	}

	m.ti.SetSuggestions(suggest)
	clilog.Writer.Debugf("Set suggestions (len: %v): %v", len(suggest), suggest)
}

// unsetAction resets the current active command/action, clears actives, and returns control to
// Mother.
func (m *Mother) unsetAction() {
	if m.active.model != nil {
		m.active.model.Reset()
	}

	m.mode = prompting
	m.active.model = nil
	m.active.command = nil
}

//#region static helper functions

// Return the parent directory to the given command
func up(dir *cobra.Command) *cobra.Command {
	if dir.Parent() == nil { // if we are at root, do nothing
		return dir
	}
	// otherwise, step upward
	clilog.Writer.Debugf("Up: %v -> %v", dir.Name(), dir.Parent().Name())
	return dir.Parent()
}

// Returns a tea.Println Cmd containing the context help for the given command.
//
// Structure:
//
// <nav> - <desc>
//
// --> <childnav> <childaction> <childnav>
//
// <nav> - <desc>
//
// --> <childaction>
//
// <action> - <desc>
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
					_, err := subchildren.WriteString(colorizer.ColorCommandName(sc) + " ")
					if err != nil {
						clilog.Writer.Warnf("Failed to generate list of subchildren: %v", err)
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

	// write help footer
	s.WriteString("\nTry " + lipgloss.NewStyle().Italic(true).Render("help help") +
		" for information on using the help command.")

	// TODO store the string within mother somewhere so we can lazy-compile all strings
	// chomp last newline and return
	return tea.Println(strings.TrimSuffix(s.String(), "\n"))
}

// Returns the present working directory, set to the primary color
func CommandPath(m *Mother) string {
	return stylesheet.PromptStyle.Render(m.pwd.CommandPath())
}

//#endregion
