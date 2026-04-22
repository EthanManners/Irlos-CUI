// SPDX-License-Identifier: GPL-3.0-or-later
package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// RenderNav returns a single-line tab bar with a clock on the right.
// activeTab is the index into tabs.
func RenderNav(width, activeTab int, tabs []string) string {
	var sb strings.Builder

	// Fill the entire line with the "off" tab background first.
	sb.WriteString(StyleTabOff.Width(width).Render(""))

	// We need to rebuild it as a single string because lipgloss styles
	// don't overlay.  Build content left-to-right.
	var content strings.Builder
	content.WriteString("  ")
	for i, name := range tabs {
		label := fmt.Sprintf("  %s  ", name)
		if i == activeTab {
			content.WriteString(StyleTabOn.Render(label))
		} else {
			content.WriteString(StyleTabOff.Render(label))
		}
		content.WriteString(" ")
	}

	clock := time.Now().Format("15:04:05")
	clockStr := StyleCyan.Render(clock)

	// Calculate remaining space for padding.
	used := lipgloss.Width(content.String())
	clockW := lipgloss.Width(clockStr)
	pad := width - used - clockW - 2
	if pad < 0 {
		pad = 0
	}
	content.WriteString(strings.Repeat(" ", pad))
	content.WriteString(clockStr)
	content.WriteString("  ")

	line := content.String()
	// Pad or truncate to exactly width.
	lw := lipgloss.Width(line)
	if lw < width {
		line += strings.Repeat(" ", width-lw)
	}
	return line
}
