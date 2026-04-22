// SPDX-License-Identifier: GPL-3.0-or-later
package ui

import (
	"strings"
)

// RenderLogs renders the Logs tab inner content.
// lines is the ring buffer; offset is the display start index; autoScroll
// when true pins the view to the tail.
func RenderLogs(width, height int, lines []string, offset int, autoScroll bool) string {
	var rows []string

	// Banner row.
	if autoScroll {
		banner := " [Auto-scroll  ↑/↓ pause  End resume] "
		rows = append(rows, StyleGreen.Render(banner))
	} else {
		banner := " [Paused  End=resume  ↑/↓ scroll] "
		rows = append(rows, StyleYellow.Bold(true).Render(banner))
	}

	innerH := height - 1 // banner takes one row

	for i := 0; i < innerH; i++ {
		li := offset + i
		if li >= len(lines) {
			rows = append(rows, "")
			continue
		}
		ln := lines[li]
		upper := strings.ToUpper(ln)

		var colored string
		switch {
		case strings.Contains(upper, "ERROR") ||
			strings.Contains(upper, "FAILED") ||
			strings.Contains(upper, "CRIT"):
			colored = StyleRed.Render(ln)
		case strings.Contains(upper, "WARN"):
			colored = StyleYellow.Render(ln)
		case strings.Contains(upper, "INFO") ||
			strings.Contains(upper, "STARTED") ||
			strings.Contains(upper, "ACTIVE"):
			colored = StyleGreen.Render(ln)
		default:
			colored = StyleDefault.Render(ln)
		}

		rows = append(rows, colored)
	}

	return joinRows(rows, height, width)
}
