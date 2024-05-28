package stylesheet

import "github.com/charmbracelet/lipgloss"

var ActionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFAAAA"))

/**
 * NOTE: Per the Lipgloss documentation (https://github.com/charmbracelet/lipgloss?tab=readme-ov-file#faq),
 * it is intelligent enough to automatically adjust or disable color depending on the given environment.
 */
