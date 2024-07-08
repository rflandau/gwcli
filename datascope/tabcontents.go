package datascope

import (
	"fmt"
	"gwcli/stylesheet"

	tea "github.com/charmbracelet/bubbletea"
)

type tab struct {
	name string

	// update and view must take a DS parameter, rather than being methods of DS, as DS's elm arch
	// passes by value
	updateFunc func(*DataScope, tea.Msg) tea.Cmd
	// see the note on updateFunc
	viewFunc func(*DataScope) string
}

const (
	results uint = iota
	help
	download
	schedule
)

// results the array of tabs with all requisite data built in
func (s *DataScope) generateTabs() []tab {
	t := make([]tab, 4)
	t[results] = tab{name: "results", updateFunc: updateResults, viewFunc: viewResults}
	t[help] = tab{
		name:       "help",
		updateFunc: func(*DataScope, tea.Msg) tea.Cmd { return nil },
		viewFunc: func(*DataScope) string {
			return fmt.Sprintf("q: return to results view\n"+
				"tab: cycle tabs\n"+
				"shift+tab: reverse cycle tab\n"+
				"%v page\n"+
				"%v scroll\n"+
				"t: toggle tabs\n"+
				"esc: quit",
				stylesheet.LeftRight, stylesheet.UpDown)
		}}
	t[download] = tab{
		name: "download",
		updateFunc: func(*DataScope, tea.Msg) tea.Cmd {
			return nil
		},
		viewFunc: func(*DataScope) string { return "" }}
	t[schedule] = tab{
		name:       "schedule",
		updateFunc: func(*DataScope, tea.Msg) tea.Cmd { return nil },
		viewFunc:   func(*DataScope) string { return "" }}

	return t
}

func updateResults(s *DataScope, msg tea.Msg) tea.Cmd {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)
	prevPage := s.pager.Page
	s.pager, cmd = s.pager.Update(msg)
	cmds = append(cmds, cmd)
	// pass the new content to the view
	s.vp.SetContent(s.displayPage())
	s.vp, cmd = s.vp.Update(msg)
	cmds = append(cmds, cmd)
	if prevPage != s.pager.Page { // if page changed, reset to top of view
		s.vp.GotoTop()
	}
	return tea.Sequence(cmds...)
}

// view when 'results' tab is active
func viewResults(s *DataScope) string {
	if !s.ready {
		return "\nInitializing..."
	}
	return fmt.Sprintf("%s\n%s", s.vp.View(), s.footer())
}
