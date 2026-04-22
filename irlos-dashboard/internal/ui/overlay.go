// SPDX-License-Identifier: GPL-3.0-or-later
package ui

import (
	"strings"
)

// PlaceOverlay centers the popup string fg onto the background string bg.
// bg must be bgW×bgH characters.  For rows touched by the popup, the
// background content in that row is replaced: the popup occupies its full
// width, and remaining columns are padded with spaces.  Rows above and below
// the popup show the original background content unchanged.
//
// This matches the Python version's behaviour of clearing the popup rectangle
// with spaces before drawing the popup box.
func PlaceOverlay(bg, fg string, bgW, bgH int) string {
	fgLines := strings.Split(fg, "\n")
	fgH := len(fgLines)

	// Measure the widest fg line (plain visible chars — fg has ANSI codes).
	fgW := 0
	for _, l := range fgLines {
		if w := VisibleWidth(l); w > fgW {
			fgW = w
		}
	}

	startY := (bgH - fgH) / 2
	startX := (bgW - fgW) / 2
	if startX < 0 {
		startX = 0
	}

	bgLines := strings.Split(bg, "\n")

	for i, fgLine := range fgLines {
		row := startY + i
		if row < 0 || row >= len(bgLines) {
			continue
		}
		// Build replacement: left_spaces + fgLine + right_spaces
		leftW := startX
		rightW := bgW - startX - fgW
		if rightW < 0 {
			rightW = 0
		}
		bgLines[row] = strings.Repeat(" ", leftW) + fgLine + strings.Repeat(" ", rightW)
	}

	return strings.Join(bgLines, "\n")
}
