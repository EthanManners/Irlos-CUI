// SPDX-License-Identifier: GPL-3.0-or-later
package ui

import (
	"fmt"
	"strings"
)

// StreamRow identifies selectable rows in the Stream tab.
const (
	StreamRowStartStop = 4 // 4 info rows (0-3), then actions
	StreamRowShell     = 5
)

// RenderStream returns the inner content of the Stream tab (without border).
// height is the inner height (box height - 2 for borders).
// width is the inner width (box width - 2 for borders).
func RenderStream(
	width, height, cursor int,
	live bool,
	uptime, scene, recvBPS, sendBPS, platform string,
	staged map[string]string,
) string {
	var rows []string

	// Big status indicator.
	var bigLabel, bigStr string
	if live {
		bigLabel = "◉  L I V E"
		bigStr = StyleGreen.Bold(true).Render(bigLabel)
	} else {
		bigLabel = "◎  O F F L I N E"
		bigStr = StyleRed.Bold(true).Render(bigLabel)
	}
	rows = append(rows, centerStr(bigStr, len(bigLabel), width))

	// Uptime.
	uptimeLbl := fmt.Sprintf("Uptime  %s", uptime)
	rows = append(rows, StyleCyan.Render(centerPad(uptimeLbl, width)))

	// Spacer.
	rows = append(rows, "")

	// Info fields.
	infoFields := [][2]string{
		{"Scene", scene},
		{"Incoming SRT", fmt.Sprintf("%s kbps", recvBPS)},
		{"Outgoing", fmt.Sprintf("%s kbps", sendBPS)},
		{"Platform", platform},
	}
	for _, kv := range infoFields {
		label := StyleYellow.Render(fmt.Sprintf("%-18s", kv[0]))
		value := StyleDefault.Render(kv[1])
		rows = append(rows, "   "+label+value)
	}

	// Separator.
	rows = append(rows, StyleBorder.Render("├"+strings.Repeat("─", width)+"┤"))

	// Actions.
	actions := []string{
		boolStr(live, "Stop Stream", "Start Stream"),
		"[Shell]",
	}
	for i, action := range actions {
		rowIdx := len(infoFields) + i
		sel := cursor == rowIdx
		var row string
		if sel {
			row = StyleSelected.Render(fmt.Sprintf("  %-*s  ", width-4, action))
		} else {
			row = StyleCyan.Bold(true).Render(fmt.Sprintf("  %-*s  ", width-4, action))
		}
		rows = append(rows, row)
	}

	return joinRows(rows, height, width)
}

func boolStr(cond bool, ifTrue, ifFalse string) string {
	if cond {
		return ifTrue
	}
	return ifFalse
}

// centerStr returns s padded symmetrically within width using visible length vlen.
func centerStr(s string, vlen, width int) string {
	pad := (width - vlen) / 2
	if pad < 0 {
		pad = 0
	}
	return strings.Repeat(" ", pad) + s
}

func centerPad(s string, width int) string {
	pad := (width - len(s)) / 2
	if pad < 0 {
		pad = 0
	}
	return strings.Repeat(" ", pad) + s
}

// joinRows concatenates rows and pads to height rows of width columns.
func joinRows(rows []string, height, width int) string {
	for len(rows) < height {
		rows = append(rows, "")
	}
	if len(rows) > height {
		rows = rows[:height]
	}
	padded := make([]string, len(rows))
	for i, r := range rows {
		padded[i] = padRight(r, width)
	}
	return strings.Join(padded, "\n")
}
