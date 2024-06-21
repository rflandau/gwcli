// Supporting code for the query actor
// Defines and supports the two, composed views in interactive mode: editor and modifiers
// Editor is as it sounds
// Modifiers provides switches and buttons to tweak how the search will be run/displayed

package query

import (
	"errors"
	"fmt"
	"gwcli/clilog"
	"gwcli/stylesheet"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

//#region editorView

// editorView represents the composable view box containing the query editor and any errors therein
type editorView struct {
	ta   textarea.Model
	err  string
	keys []key.Binding
}

func initialEdiorView(height, width uint) editorView {
	ev := editorView{}

	// configure text area
	ev.ta = textarea.New()
	ev.ta.ShowLineNumbers = true
	ev.ta.Prompt = stylesheet.PromptPrefix
	ev.ta.SetWidth(int(width))
	ev.ta.SetHeight(int(height))
	ev.ta.Focus()
	// set up the help keys
	ev.keys = []key.Binding{ // 0: submit
		key.NewBinding(
			key.WithKeys("alt+enter"),
			key.WithHelp("alt+enter", "submit query"),
		)}

	return ev
}

func (ev *editorView) update(msg tea.Msg) (cmd tea.Cmd, submit bool) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		ev.err = ""
		switch {
		case key.Matches(msg, ev.keys[0]): // submit
			if ev.ta.Value() == "" {
				// superfluous request
				ev.err = "empty request"
				// falls through to standard update
			} else {
				return nil, true
			}
		}
	}
	var t tea.Cmd
	ev.ta, t = ev.ta.Update(msg)
	return t, false
}

func (va *editorView) view() string {
	return fmt.Sprintf("%s\n%s\n%s", stylesheet.Header1Style.Render("Query:"), va.ta.View(),
		stylesheet.ErrStyle.Width(stylesheet.TIWidth).Render(va.err)) // add a width style for wrapping
}

//#endregion editorView

//#region modifView

type modifSelection = uint

const (
	lowBound modifSelection = iota
	duration
	outFile
	appendToFile
	highBound
)

const defaultModifSelection = duration

// modifView represents the composable view box containing all configurable features of the query
type modifView struct {
	width    uint
	height   uint
	selected uint // tracks which modifier is currently active w/in this view
	// knobs available to user
	durationTI   textinput.Model
	outfileTI    textinput.Model
	appendToFile bool
	json         bool
	csv          bool

	keys []key.Binding
}

// generate the second view to be composed with the query editor
func initialModifView(height, width uint) modifView {

	mv := modifView{
		width:    width,
		height:   height,
		selected: defaultModifSelection,
		keys: []key.Binding{
			key.NewBinding(
				key.WithKeys("↑/↓"),
				key.WithHelp("↑/↓", "select input"),
			)},
	}

	// build duration ti
	mv.durationTI = textinput.New()
	mv.durationTI.Width = int(width)
	mv.durationTI.Blur()
	mv.durationTI.Prompt = ""
	mv.durationTI.SetValue(defaultDuration.String())
	mv.durationTI.Placeholder = "1h00m00s00ms00us00ns"
	mv.durationTI.Validate = func(s string) error {
		// checks that the string is composed of valid characters for duration parsing
		// (0-9 and h,m,s,u,n)
		// ! does not confirm that it is a valid duration!
		validChars := map[rune]interface{}{
			'h': nil, 'm': nil, 's': nil,
			'u': nil, 'n': nil, '.': nil,
		}
		for _, r := range s {
			if unicode.IsDigit(r) {
				continue
			}
			if _, f := validChars[r]; !f {
				return errors.New("only digits or the characters h, m, s, u, and n are allowed")
			}
		}
		return nil
	}

	// build outFile ti
	mv.outfileTI = textinput.New()
	mv.outfileTI.Width = int(width)
	mv.outfileTI.Blur()
	mv.outfileTI.Placeholder = "(optional)"
	mv.outfileTI.Prompt = ""

	return mv

}

// Unfocuses this view, blurring all text inputs
func (mv *modifView) blur() {
	mv.durationTI.Blur()
	mv.outfileTI.Blur()
}

func (mv *modifView) update(msg tea.Msg) []tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp:
			mv.selected -= 1
			if mv.selected <= lowBound {
				mv.selected = highBound - 1
			}
			mv.focusSelected()
		case tea.KeyDown:
			mv.selected += 1
			if mv.selected >= highBound {
				mv.selected = lowBound + 1
			}
			mv.focusSelected()
		case tea.KeySpace, tea.KeyEnter:
			if mv.selected == appendToFile {
				mv.appendToFile = !mv.appendToFile
			}
		}
	}
	var cmds []tea.Cmd = []tea.Cmd{}
	var t tea.Cmd
	mv.durationTI, t = mv.durationTI.Update(msg)
	if t != nil {
		cmds = append(cmds, t)
	}
	mv.outfileTI, t = mv.outfileTI.Update(msg)
	if t != nil {
		cmds = append(cmds, t)
	}

	return cmds
}

// Focuses the text input associated with the current selection, blurring all others
func (mv *modifView) focusSelected() {
	switch mv.selected {
	case duration:
		mv.durationTI.Focus()
		mv.outfileTI.Blur()
	case outFile:
		mv.durationTI.Blur()
		mv.outfileTI.Focus()
	case appendToFile:
		mv.durationTI.Blur()
		mv.outfileTI.Blur()
	default:
		clilog.Writer.Errorf("Failed to update modifier view focus: unknown selected field %d",
			mv.selected)
	}
}

func (mv *modifView) view() string {
	var bldr strings.Builder

	bldr.WriteString(stylesheet.Header1Style.Render("Duration:") + "\n")
	bldr.WriteString(
		fmt.Sprintf("%c %s\n", pip(mv.selected, duration), mv.durationTI.View()),
	)

	bldr.WriteString(stylesheet.Header1Style.Render("Output Path:") + "\n")
	bldr.WriteString(
		fmt.Sprintf("%c %s\n", pip(mv.selected, outFile), mv.outfileTI.View()),
	)
	var checkmark rune = '☐'
	if mv.appendToFile {
		checkmark = '☑'
	}
	// if the outfile field is empty, render append in grey
	if strings.TrimSpace(mv.outfileTI.Value()) == "" {
		bldr.WriteString(
			fmt.Sprintf("%c %s\n",
				pip(mv.selected, appendToFile),
				stylesheet.GreyedOutStyle.Render(string(checkmark)+" "+"Append?")),
		)
	} else {
		bldr.WriteString(
			fmt.Sprintf("%c %c %s\n",
				pip(mv.selected, appendToFile),
				checkmark, stylesheet.Header1Style.Render("Append?")),
		)
	}

	return bldr.String()
}

// if this field is the selected field, returns the selection rune.
// otherwise, returns a space
func pip(selected, field uint) rune {
	if selected == field {
		return stylesheet.SelectionPrefix
	}
	return ' '
}

//#endregion modifView