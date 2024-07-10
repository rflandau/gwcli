// colorizer provides common utilities that rely on the stylesheet.
// These functions are to support a consistent UI.
package colorizer

import (
	"fmt"
	"gwcli/action"
	"gwcli/stylesheet"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// tea.Printf wrapper that colors the output as an error
func ErrPrintf(format string, a ...interface{}) tea.Cmd {
	return tea.Printf(stylesheet.ErrStyle.Render(fmt.Sprintf(format, a...)))
}

// Given a command, returns its name appropriately colored by its group (action or nav).
// Defaults to nav color.
func ColorCommandName(c *cobra.Command) string {
	if action.Is(c) {
		return stylesheet.ActionStyle.Render(c.Name())
	} else {
		return stylesheet.NavStyle.Render(c.Name())
	}
}

func Pip(selected, field uint) string {
	if selected == field {
		return lipgloss.NewStyle().Foreground(stylesheet.AccentColor2).Render(string(stylesheet.SelectionPrefix))
	}
	return ""
}
