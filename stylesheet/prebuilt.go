package stylesheet

import "github.com/charmbracelet/bubbles/textinput"

/**
 * Prebuilt, commonly-used models for stylistic consistency.
 */

// Creates a textinput with common attributes.
func NewTI(defVal string) textinput.Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Width = 20
	ti.SetValue(defVal)
	return ti
}
