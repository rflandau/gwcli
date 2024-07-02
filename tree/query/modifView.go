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

/**
 * when adding new items to the view, make sure to:
 * 1: add a new modfSelection constant
 * 2: add enter/space/selection functionality to the update function
 * 3: if it is a text input:
 *		a: initialize it!
 *		b: include it in focusedSelected()
 *		c: call its update function in Update()
 * 4: clear it in .Reset()
 * 5: if it has an associated flag, make sure it is set in `actor.go`.SetArgs()
 */
type modifSelection = uint

const (
	lowBound modifSelection = iota
	duration
	outFile
	appendToFile
	json
	csv
	nohistory
	scheduled
	name
	desc
	cronfreq
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
	schedule     struct {
		enabled    bool
		nameTI     textinput.Model
		descTI     textinput.Model
		cronfreqTI textinput.Model
	}

	keys []key.Binding
}

// generate the second view to be composed with the query editor
func initialModifView(height, width uint) modifView {

	// helper function for setting up a standard text input
	newTI := func(dflt string) textinput.Model {
		ti := textinput.New()
		ti.Width = int(width)
		ti.Blur()
		ti.Prompt = ""
		ti.SetValue(dflt)
		return ti
	}

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
	mv.durationTI = newTI(defaultDuration.String())
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
	mv.outfileTI = newTI("")
	mv.outfileTI.Placeholder = "(optional)"

	// build name ti
	mv.schedule.nameTI = newTI("")

	// build description ti
	mv.schedule.descTI = newTI("")

	// buid schedule frequency ti
	mv.schedule.cronfreqTI = newTI("")
	mv.schedule.cronfreqTI.Placeholder = "* * * * *"
	mv.schedule.cronfreqTI.Validate = func(s string) error {
		exploded := strings.Split(s, " ")
		if len(exploded) > 5 {
			return errors.New("must be exactly 5 values")
		}
		return nil
	}

	return mv

}

// Unfocuses this view, blurring all text inputs
func (mv *modifView) blur() {
	mv.durationTI.Blur()
	mv.outfileTI.Blur()
	mv.schedule.nameTI.Blur()
	mv.schedule.descTI.Blur()
	mv.schedule.cronfreqTI.Blur()
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
				return nil
			case json:
				mv.json = !mv.json
				if mv.json {
					mv.csv = false
				}
				return nil
			case csv:
				mv.csv = !mv.csv
				if mv.csv {
					mv.json = false
				}
				return nil
			case nohistory:
				mv.nohistory = !mv.nohistory
				return nil
			case scheduled:
				mv.schedule.enabled = !mv.schedule.enabled
				return nil
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
	mv.schedule.nameTI, t = mv.schedule.nameTI.Update(msg)
	if t != nil {
		cmds = append(cmds, t)
	}
	mv.schedule.descTI, t = mv.schedule.descTI.Update(msg)
	if t != nil {
		cmds = append(cmds, t)
	}
	mv.schedule.cronfreqTI, t = mv.schedule.cronfreqTI.Update(msg)
	if t != nil {
		cmds = append(cmds, t)
	}

	return cmds
}

// Focuses the text input associated with the current selection, blurring all others
func (mv *modifView) focusSelected() {
	mv.blur()

	switch mv.selected {
	case duration:
		mv.durationTI.Focus()
	case outFile:
		mv.outfileTI.Focus()
	case name:
		mv.schedule.nameTI.Focus()
	case desc:
		mv.schedule.descTI.Focus()
	case cronfreq:
		mv.schedule.cronfreqTI.Focus()
	}
}

func (mv *modifView) view() string {
	// TODO need to rework the look of modifView to make dependent fields clearer
	var bldr strings.Builder

	bldr.WriteString(" " + stylesheet.Header1Style.Render("Duration:") + "\n")
	bldr.WriteString(
		fmt.Sprintf("%c%s\n", pip(mv.selected, duration), mv.durationTI.View()),
	)

	bldr.WriteString(drawOutpathSection(mv, !mv.schedule.enabled))

	bldr.WriteString(viewBool(mv.selected, nohistory, mv.nohistory, "Exclude from History?",
		stylesheet.Header1Style, false))

	bldr.WriteString(drawScheduleSection(mv))

	return bldr.String()
}

// Generates a string representing all output path modifiers and fields.
// This whole section is greyed out if !enabled
func drawOutpathSection(mv *modifView, enabled bool) string {
	var (
		b               strings.Builder
		outpathTitleSty lipgloss.Style = stylesheet.Header1Style
		outpathTISty    lipgloss.Style = lipgloss.NewStyle()
	)
	if !enabled {
		outpathTitleSty = stylesheet.GreyedOutStyle
		outpathTISty = stylesheet.GreyedOutStyle
	}
	b.WriteString(" " + outpathTitleSty.Render("Output Path:") + "\n")
	b.WriteString(
		fmt.Sprintf("%c%s\n", pip(mv.selected, outFile), outpathTISty.Render(mv.outfileTI.View())),
	)

	// grey out outpath options if outpath is empty
	if strings.TrimSpace(mv.outfileTI.Value()) == "" {
		outpathTitleSty = stylesheet.GreyedOutStyle
	}

	b.WriteString(stylesheet.Indent +
		viewBool(mv.selected, appendToFile, mv.appendToFile, "Append?", outpathTitleSty, true))
	b.WriteString(stylesheet.Indent +
		viewBool(mv.selected, json, mv.json, "JSON", outpathTitleSty, true))
	b.WriteString(stylesheet.Indent +
		viewBool(mv.selected, csv, mv.csv, "CSV", outpathTitleSty, true))

	return b.String()
}

func drawScheduleSection(mv *modifView) string {
	var (
		b           strings.Builder
		schTitleSty lipgloss.Style = stylesheet.Header1Style
		schTISty    lipgloss.Style = lipgloss.NewStyle()
	)
	// bool to enable the rest of the section
	b.WriteString(viewBool(mv.selected, scheduled, mv.schedule.enabled, "Schedule?",
		schTitleSty, false))

	if !mv.schedule.enabled { // only display rest of section if scheduled
		schTitleSty = stylesheet.GreyedOutStyle
		schTISty = stylesheet.GreyedOutStyle
	}

	b.WriteString(stylesheet.Indent + " | " + schTitleSty.Render("Name:") + "\n")
	b.WriteString(
		fmt.Sprintf(stylesheet.Indent+"%c| %s\n",
			pip(mv.selected, name), schTISty.Render(mv.schedule.nameTI.View())),
	)
	b.WriteString(stylesheet.Indent + " | " + schTitleSty.Render("Desc:") + "\n")
	b.WriteString(
		fmt.Sprintf(stylesheet.Indent+"%c| %s\n",
			pip(mv.selected, desc), schTISty.Render(mv.schedule.descTI.View())),
	)
	b.WriteString(stylesheet.Indent + " | " + schTitleSty.Render("Schedule:") + "\n")
	b.WriteString(
		fmt.Sprintf(stylesheet.Indent+"%c| %s\n",
			pip(mv.selected, cronfreq), schTISty.Render(mv.schedule.cronfreqTI.View())),
	)
	return b.String()
}

func (mv *modifView) reset() {
	mv.selected = defaultModifSelection
	mv.durationTI.Reset()
	mv.outfileTI.Reset()
	mv.blur()
	mv.appendToFile = false
	mv.json = false
	mv.csv = false
	mv.nohistory = false
	mv.schedule.enabled = false
	mv.schedule.nameTI.Reset()
	mv.schedule.descTI.Reset()
	mv.schedule.cronfreqTI.Reset()

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
