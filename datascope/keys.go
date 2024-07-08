package datascope

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

var showTabsKey key.Binding = key.NewBinding(
	key.WithKeys(tea.KeyCtrlS.String()),
)
