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
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type modifSelection = uint

const (
	lowBound modifSelection = iota
	duration
	outFile
	appendToFile
	json
	csv
	nohistory
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
	nohistory    bool

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
				key.WithKeys(stylesheet.UpDown),
				key.WithHelp(stylesheet.UpDown, "select input"),
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
			switch mv.selected {
			case appendToFile:
				mv.appendToFile = !mv.appendToFile
			case json:
				mv.json = !mv.json
				if mv.json {
					mv.csv = false
				}
			case csv:
				mv.csv = !mv.csv
				if mv.csv {
					mv.json = false
				}
			case nohistory:
				mv.nohistory = !mv.nohistory
			}
			return nil
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
	case appendToFile, json, csv, nohistory:
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

	// view boolean switches
	// first three depend on outfile
	bldr.WriteString(viewBool(pip(mv.selected, appendToFile),
		mv.appendToFile, "Append?", mv.outfileTI))
	bldr.WriteString(viewBool(pip(mv.selected, json),
		mv.json, "JSON", mv.outfileTI))
	bldr.WriteString(viewBool(pip(mv.selected, csv),
		mv.csv, "CSV", mv.outfileTI))

	bldr.WriteString(viewBool(pip(mv.selected, nohistory),
		mv.nohistory, "No History", nil))
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

type dependsOn interface {
	Value() string
}

// Returns a string representing the current state of the given boolean value.
// DependsOn is any struct with a .Value that can be checked for emptiness.
func viewBool(pip rune, val bool, fieldName string, dependsOn dependsOn) string {
	var sty lipgloss.Style = stylesheet.Header1Style
	// if depended value is given and empty, grey out
	if dependsOn != nil && dependsOn.Value() == "" {
		sty = stylesheet.GreyedOutStyle
	}

	var checked rune = ' '
	if val {
		checked = '✓'
	}
	return fmt.Sprintf("%c [%s] %s\n", pip, sty.Render(string(checked)), sty.Render(fieldName))
}
