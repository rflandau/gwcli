package stylesheet

import (
	"fmt"
	"gwcli/action"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// tea.Printf wrapper that colors the output as an error
func ErrPrintf(format string, a ...interface{}) tea.Cmd {
	return tea.Printf(ErrStyle.Render(fmt.Sprintf(format, a...)))
}

// Given a command, returns its name appropriately colored by its group (action or nav).
// Defaults to nav color.
func ColorCommandName(c *cobra.Command) string {
	if action.Is(c) {
		return ActionStyle.Render(c.Name())
	} else {
		return NavStyle.Render(c.Name())
	}
}
