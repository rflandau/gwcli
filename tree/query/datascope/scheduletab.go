package datascope

import (
	"errors"
	"fmt"
	"gwcli/clilog"
	"gwcli/connection"
	"gwcli/stylesheet"
	"gwcli/stylesheet/colorizer"
	"gwcli/utilities/uniques"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type scheduleCursor = uint

const (
	schlowBound scheduleCursor = iota
	schcronfreq
	schname
	schdesc
	schhighBound
)

type scheduleTab struct {
	selected         scheduleCursor
	resultString     string // result of the previous scheduling
	inputErrorString string // issues with current user input

	cronfreqTI textinput.Model
	nameTI     textinput.Model
	descTI     textinput.Model
}

func initScheduleTab() scheduleTab {
	sch := scheduleTab{
		cronfreqTI: stylesheet.NewTI(""),
		nameTI:     stylesheet.NewTI(""),
		descTI:     stylesheet.NewTI(""),
	}

	// set TI-specific options
	sch.cronfreqTI.Placeholder = "* * * * *"
	sch.cronfreqTI.Validate = func(s string) error {
		// check for an empty TI
		if strings.TrimSpace(s) == "" {
			return nil
		}
		runes := []rune(s)
		if len(runes) < 1 {
			return nil
		}

		// check that the latest input is a digit or space
		if char := runes[len(runes)-1]; !unicode.IsSpace(char) && !unicode.IsDigit(rune(char)) {
			return errors.New("frequency can contain only digits")
		}

		// check that we do not have too many values
		exploded := strings.Split(s, " ")
		if len(exploded) > 5 {
			return errors.New("must be exactly 5 values")
		}

		// check that the newest word is <= 2 characters
		lastWord := []rune(exploded[len(exploded)-1])
		if len(lastWord) > 2 {
			return errors.New("each word is <= 2 digits")
		}

		// checking for the values of each word is delayed until connection.CreateScheduledSearch to
		// save on cycles

		return nil
	}

	// focus frequency by default
	sch.cronfreqTI.Focus()
	sch.selected = schcronfreq

	return sch
}

func updateSchedule(s *DataScope, msg tea.Msg) tea.Cmd {
	if msg, ok := msg.(tea.KeyMsg); ok {
		s.schedule.inputErrorString = ""
		switch msg.Type {
		case tea.KeyUp:
			s.schedule.selected -= 1
			if s.schedule.selected <= schlowBound {
				s.schedule.selected = schhighBound - 1
			}
			s.schedule.focusSelected()
			return textinput.Blink
		case tea.KeyDown:
			s.schedule.selected += 1
			if s.schedule.selected >= schhighBound {
				s.schedule.selected = schlowBound + 1
			}
			s.schedule.focusSelected()
			return textinput.Blink
		case tea.KeyEnter:
			if msg.Alt { // only accept alt+enter
				// gather and validate selections
				var (
					n   = strings.TrimSpace(s.schedule.nameTI.Value())
					d   = strings.TrimSpace(s.schedule.descTI.Value())
					cf  = strings.TrimSpace(s.schedule.cronfreqTI.Value())
					qry = s.search.SearchString
				)
				// fetch the duration from the search struct
				start, err := time.Parse(uniques.SearchTimeFormat, s.search.SearchStart)
				if err != nil {
					s.schedule.resultString = "failed to read duration start time: " + err.Error()
					clilog.Writer.Error(s.schedule.resultString)
					return nil
				}
				end, err := time.Parse(uniques.SearchTimeFormat, s.search.SearchEnd)
				if err != nil {
					s.schedule.resultString = "failed to read duration end time: " + err.Error()
					clilog.Writer.Error(s.schedule.resultString)
					return nil
				}

				id, invalid, err := connection.CreateScheduledSearch(n, d, cf, qry, end.Sub(start))
				if invalid != "" { // bad parameters
					s.schedule.inputErrorString = invalid
					clilog.Writer.Debug(s.schedule.inputErrorString)
				} else if err != nil {
					s.schedule.resultString = err.Error()
					clilog.Writer.Error(err.Error())
				} else {
					s.schedule.resultString = fmt.Sprintf("successfully scheduled query (ID: %v)", id)
					clilog.Writer.Info(s.schedule.resultString)
				}
				return nil
			}
		}
	}

	// pass onto the TIs
	var cmds []tea.Cmd = make([]tea.Cmd, 3)
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		s.schedule.cronfreqTI, cmds[0] = s.schedule.cronfreqTI.Update(msg)
	}()
	go func() {
		defer wg.Done()
		s.schedule.nameTI, cmds[1] = s.schedule.nameTI.Update(msg)
	}()
	go func() {
		defer wg.Done()
		s.schedule.descTI, cmds[2] = s.schedule.descTI.Update(msg)
	}()

	wg.Wait()

	return tea.Batch(cmds...)
}

func viewSchedule(s *DataScope) string {
	sel := s.schedule.selected // brevity

	var (
		titleSty       lipgloss.Style = stylesheet.Header1Style
		leftAlignerSty lipgloss.Style = lipgloss.NewStyle().
				Width(20).
				AlignHorizontal(lipgloss.Right).
				PaddingRight(1)
	)

	tabDesc := tabDescStyle(s.vp.Width).Render("Schedule this search to be rerun at" +
		" consistent intervals." + "\nQuery: " + stylesheet.Header2Style.Render(s.search.SearchString))

	// build the field names column
	fields := lipgloss.JoinVertical(lipgloss.Right,
		leftAlignerSty.Render(fmt.Sprintf("%s%s",
			colorizer.Pip(sel, schcronfreq), titleSty.Render("Frequency:"))),
		leftAlignerSty.Render(fmt.Sprintf("%s%s",
			colorizer.Pip(sel, schname), titleSty.Render("Name:"))),
		leftAlignerSty.Render(fmt.Sprintf("%s%s",
			colorizer.Pip(sel, schdesc), titleSty.Render("Description:"))),
	)

	// build the TIs column
	TIs := lipgloss.JoinVertical(lipgloss.Left,
		s.schedule.cronfreqTI.View(),
		s.schedule.nameTI.View(),
		s.schedule.descTI.View(),
	)

	composed := lipgloss.JoinHorizontal(lipgloss.Center,
		fields,
		TIs)

	return lipgloss.Place(s.vp.Width, s.vp.Height,
		lipgloss.Center, verticalPlace,
		lipgloss.JoinVertical(lipgloss.Center,
			tabDesc,
			composed,
			"",
			submitString(s.schedule.inputErrorString, s.schedule.resultString),
		),
	)
}

// Focuses the TI corresponding to sch.selected and blurs all others.
func (sch *scheduleTab) focusSelected() {
	switch sch.selected {
	case schcronfreq:
		sch.cronfreqTI.Focus()
		sch.nameTI.Blur()
		sch.descTI.Blur()
	case schname:
		sch.cronfreqTI.Blur()
		sch.nameTI.Focus()
		sch.descTI.Blur()
	case schdesc:
		sch.cronfreqTI.Blur()
		sch.nameTI.Blur()
		sch.descTI.Focus()
	}
}
