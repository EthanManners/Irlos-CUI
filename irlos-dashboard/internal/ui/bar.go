// SPDX-License-Identifier: GPL-3.0-or-later
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Bar renders a colour-coded progress bar into a single line string.
//
//   - pct: 0.0–1.0
//   - width: number of block characters in the track
//   - label: left-padded to 12 chars (omit with "")
//   - rightText: appended after the percentage (omit with "")
func Bar(pct float64, width int, label, rightText string) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 1 {
		pct = 1
	}

	filled := int(pct * float64(width))
	empty := width - filled

	var fillStyle lipgloss.Style
	switch {
	case pct >= 0.90:
		fillStyle = StyleBarCrit
	case pct >= 0.70:
		fillStyle = StyleBarWarn
	default:
		fillStyle = StyleBarOK
	}

	var sb strings.Builder

	if label != "" {
		sb.WriteString(StyleYellow.Render(fmt.Sprintf("%-12s", label)))
	}

	if filled > 0 {
		sb.WriteString(fillStyle.Render(strings.Repeat(" ", filled)))
	}
	if empty > 0 {
		sb.WriteString(StyleBarDim.Render(strings.Repeat("░", empty)))
	}

	sb.WriteString(StyleDefault.Render(fmt.Sprintf(" %5.1f%%", pct*100)))

	if rightText != "" {
		sb.WriteString("  ")
		sb.WriteString(StyleCyan.Render(rightText))
	}

	return sb.String()
}

// Dot renders a coloured status dot with an optional label.
// active → green ●, inactive → red ●
func Dot(active bool, label string) string {
	dotStyle := StyleRed.Bold(true)
	if active {
		dotStyle = StyleGreen.Bold(true)
	}
	s := dotStyle.Render("●")
	if label != "" {
		s += " " + StyleDefault.Render(label)
	}
	return s
}
