// Supporting code for the query actor
// Modifiers provides switches and buttons to tweak how the search will be run/displayed

package query

import (
	"errors"
	"fmt"
	"gwcli/stylesheet"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// modifSelection provides the skeleton for cursoring through options within this view.
// All other options have been relocated so it is rather overengineered currently.
// However, its skeleton has been left in place so adding new options in the future is easy.
// See datascope's download and schedule tabs for examples
type modifSelection = uint

const (
	lowBound modifSelection = iota
	duration
	highBound
)

// modifView represents the composable view box containing all configurable features of the query
type modifView struct {
	width    uint
	height   uint
	selected uint // tracks which modifier is currently active w/in this view
	// knobs available to user
	durationTI textinput.Model

	keys []key.Binding
}

// generate the second view to be composed with the query editor
func initialModifView(height, width uint) modifView {
	mv := modifView{
		width:    width,
		height:   height,
		selected: duration, // default to duration
		keys: []key.Binding{
			key.NewBinding(
				key.WithKeys(stylesheet.UpDown),
				// help is not necessary when there is only one option
				// key.WithHelp(stylesheet.UpDown, "select input"),
			)},
	}

	// build duration ti
	mv.durationTI = stylesheet.NewTI(defaultDuration.String())
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
			if _, found := validChars[r]; !found {
				return errors.New("only digits or the characters h, m, s, u, and n are allowed")
			}
		}
		return nil
	}
	return mv
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
		case tea.KeyDown:
			mv.selected += 1
			if mv.selected >= highBound {
				mv.selected = lowBound + 1
			}
		}
	}
	var cmds []tea.Cmd = make([]tea.Cmd, 1)
	mv.durationTI, cmds[0] = mv.durationTI.Update(msg)

	return cmds
}

func (mv *modifView) view() string {
	var bldr strings.Builder

	bldr.WriteString(" " + stylesheet.Header1Style.Render("Duration:") + "\n")
	bldr.WriteString(
		fmt.Sprintf("%c%s\n", pip(mv.selected, duration), mv.durationTI.View()),
	)

	return bldr.String()
}

func (mv *modifView) reset() {
	mv.durationTI.Reset()
	mv.durationTI.Blur()
}

// if this field is the selected field, returns the selection rune.
// otherwise, returns a space
func pip(selected, field uint) rune {
	if selected == field {
		return stylesheet.SelectionPrefix
	}
	return ' '
}

// Returns a string representing the current state of the given boolean value.
func viewBool(selected uint, field uint, val bool, fieldName string, sty lipgloss.Style, child bool) string {
	var checked rune = ' '
	if val {
		checked = '✓'
	}

	var divisor = ""
	if child {
		divisor = "| "
	}

	return fmt.Sprintf("%c%v[%s] %s\n", pip(selected, field), divisor, sty.Render(string(checked)), sty.Render(fieldName))
}
