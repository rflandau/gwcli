package datascope

import (
	"errors"
	"fmt"
	"gwcli/stylesheet"
	"strings"
	"sync"

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
	selected scheduleCursor

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
		exploded := strings.Split(s, " ")
		if len(exploded) > 5 {
			return errors.New("must be exactly 5 values")
		}
		return nil
	}

	// focus frequency by default
	sch.cronfreqTI.Focus()
	sch.selected = schcronfreq

	return sch
}

func updateSchedule(s *DataScope, msg tea.Msg) tea.Cmd {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.Type {
		case tea.KeyUp:
			s.schedule.selected -= 1
			if s.schedule.selected <= schlowBound {
				s.schedule.selected = schhighBound - 1
			}
			s.schedule.focusSelected()
		case tea.KeyDown:
			s.schedule.selected += 1
			if s.schedule.selected >= schhighBound {
				s.schedule.selected = schlowBound + 1
			}
			s.schedule.focusSelected()
		case tea.KeyEnter:
			if msg.Alt { // only accept alt+enter
				// TODO submit scheduled query
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

	var titleSty lipgloss.Style = stylesheet.Header1Style.Width(20).AlignHorizontal(lipgloss.Right)

	// build the field names column
	fields := lipgloss.JoinVertical(lipgloss.Right,
		fmt.Sprintf("%c%s ", pip(sel, schcronfreq), titleSty.Render("Frequency: ")),
		fmt.Sprintf("%c%s ", pip(sel, schname), titleSty.Render("Name: ")),
		fmt.Sprintf("%c%s ", pip(sel, schdesc), titleSty.Render("Description: ")),
	)

	// build the TIs column
	TIs := lipgloss.JoinVertical(lipgloss.Left,
		s.schedule.cronfreqTI.View(),
		s.schedule.nameTI.View(),
		s.schedule.descTI.View(),
	)

	// build each TI
	/*cf := fmt.Sprintf("%c%s: %s",
		pip(sel, schcronfreq),
		titleSty.Render("Frequency: "),
		s.schedule.cronfreqTI.View())
	name := fmt.Sprintf("%c%s: %s",
		pip(sel, schname),
		titleSty.Render("Name: "),
		s.schedule.nameTI.View())
	desc := fmt.Sprintf("%c%s: %s",
		pip(sel, schdesc),
		titleSty.Render("Description: "),
		s.schedule.descTI.View())

	options := lipgloss.JoinVertical(lipgloss.Center,
		cf,
		name,
		desc)*/
	return lipgloss.Place(s.vp.Width, s.vp.Height, lipgloss.Center, lipgloss.Center, lipgloss.JoinHorizontal(lipgloss.Center, fields, TIs))
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