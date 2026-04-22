// SPDX-License-Identifier: GPL-3.0-or-later
package ui

import (
	"strings"
)

// wordmark is the ASCII art IRLOS logo (6 lines).
var wordmark = []string{
	" ██╗██████╗ ██╗      ██████╗ ███████╗",
	" ██║██╔══██╗██║     ██╔═══██╗██╔════╝",
	" ██║██████╔╝██║     ██║   ██║███████╗",
	" ██║██╔══██╗██║     ██║   ██║╚════██║",
	" ██║██║  ██║███████╗╚██████╔╝███████║",
	" ╚═╝╚═╝  ╚═╝╚══════╝ ╚═════╝ ╚══════╝",
}

// wordmarkWidth is the visual width of the widest wordmark line.
const wordmarkWidth = 38

// WordmarkHeight is the number of lines in the header block:
// 6 wordmark lines + 1 blank line below.
const WordmarkHeight = 7

// RenderHeader returns the wordmark + live badge block as a slice of lines.
// Total height is WordmarkHeight (7 lines).
func RenderHeader(width int, live bool) string {
	cx := (width - wordmarkWidth) / 2
	if cx < 0 {
		cx = 0
	}
	leftPad := strings.Repeat(" ", cx)

	var badge, badgeStr string
	if live {
		badge = "● LIVE"
		badgeStr = StyleGreen.Bold(true).Render(badge)
	} else {
		badge = "○ OFFLINE"
		badgeStr = StyleRed.Bold(true).Render(badge)
	}

	// Badge position: right of the wordmark on row 2 (0-indexed), or below.
	badgeX := cx + wordmarkWidth + 3
	inlineOK := badgeX+len(badge) < width

	var lines []string
	for i, wl := range wordmark {
		line := leftPad + StyleCyan.Bold(true).Render(wl)
		if i == 2 && inlineOK {
			// Append badge inline on this row.
			lineW := cx + wordmarkWidth
			pad := badgeX - lineW
			if pad < 0 {
				pad = 0
			}
			line += strings.Repeat(" ", pad) + badgeStr
		}
		lines = append(lines, line)
	}

	if !inlineOK {
		// Badge on a separate line below the wordmark.
		bc := (width - len(badge)) / 2
		if bc < 0 {
			bc = 0
		}
		lines = append(lines, strings.Repeat(" ", bc)+badgeStr)
	} else {
		lines = append(lines, "") // blank line separator
	}

	return strings.Join(lines, "\n")
}
