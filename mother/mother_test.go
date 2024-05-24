package mother

import (
	"io"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
)

/**
 * Tests that New leaves no fields unfilled
 */
func TestFieldsPopulated(t *testing.T) {
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.TrueColor)
	r.SetHasDarkBackground(true)
	m := new(&cobra.Command{}, r)

	if m.root == nil {
		t.Error("root is nil")
	}
	if m.pwd == nil {
		t.Error("root is nil")
	}

	// TODO this test needs to be expanded to be run with equal renderers to
	// ensure styling is being discarded only when required
	if m.style.nav.Render("text") != lipgloss.NewStyle().Render("text") {
		t.Error("nav style is not bare")
	}
	if m.style.action.Render("text") != lipgloss.NewStyle().Render("text") {
		t.Error("action style is bare")
	}
	if m.style.error.Render("text") != lipgloss.NewStyle().Render("text") {
		t.Error("error style is bare")
	}

	// Cannot compare against a bare textinput.Model to check that New called textinput.Input
	if !m.ti.Focused() {
		t.Error("text input prompt is not focused")
	}
}
