// SPDX-License-Identifier: GPL-3.0-or-later
// Package ui contains pure rendering functions for the dashboard.
// Functions here are stateless: they accept data as parameters and return
// styled strings.  They do NOT import the model package.
package ui

import "github.com/charmbracelet/lipgloss"

// Terminal colour indices (256-colour / ANSI).
const (
	colorBlack  = lipgloss.Color("0")
	colorRed    = lipgloss.Color("9")
	colorGreen  = lipgloss.Color("10")
	colorYellow = lipgloss.Color("11")
	colorBlue   = lipgloss.Color("12")
	colorCyan   = lipgloss.Color("14")
	colorWhite  = lipgloss.Color("15")
	colorGray   = lipgloss.Color("8") // dim / dark gray
)

var (
	StyleDefault = lipgloss.NewStyle().Foreground(colorWhite)
	StyleCyan    = lipgloss.NewStyle().Foreground(colorCyan)
	StyleGreen   = lipgloss.NewStyle().Foreground(colorGreen)
	StyleRed     = lipgloss.NewStyle().Foreground(colorRed)
	StyleYellow  = lipgloss.NewStyle().Foreground(colorYellow)
	StyleDim     = lipgloss.NewStyle().Foreground(colorGray)
	StyleBold    = lipgloss.NewStyle().Bold(true)

	StyleTabOn = lipgloss.NewStyle().
			Foreground(colorBlack).
			Background(colorCyan).
			Bold(true)

	StyleTabOff = lipgloss.NewStyle().
			Foreground(colorWhite)

	StyleSelected = lipgloss.NewStyle().
			Foreground(colorBlack).
			Background(colorWhite)

	StyleStaged = lipgloss.NewStyle().
			Foreground(colorYellow).
			Bold(true)

	StyleBorder = lipgloss.NewStyle().Foreground(colorCyan)

	StyleBarOK   = lipgloss.NewStyle().Background(colorGreen).Foreground(colorGreen)
	StyleBarWarn = lipgloss.NewStyle().Background(colorYellow).Foreground(colorYellow)
	StyleBarCrit = lipgloss.NewStyle().Background(colorRed).Foreground(colorRed)
	StyleBarDim  = lipgloss.NewStyle().Foreground(colorGray)
)
