// SPDX-License-Identifier: GPL-3.0-or-later
package ui

import (
	"fmt"
	"strings"
)

// RenderVNC renders the VNC tab inner content.
func RenderVNC(width, height, cursor int, vncUp, novncUp bool, ip string) string {
	var rows []string

	rows = append(rows, "", "") // top padding

	// VNC service status row.
	vncDot := Dot(vncUp, boolStr(vncUp, "Running", "Stopped"))
	rows = append(rows, "   "+StyleYellow.Render(fmt.Sprintf("%-17s", "VNC Service"))+"  "+vncDot)

	rows = append(rows, "")

	// SSH tunnel command.
	rows = append(rows, "   "+StyleYellow.Bold(true).Render("SSH Tunnel Command"))
	cmd := fmt.Sprintf("ssh -L 5900:%s:5900 irlos@%s", ip, ip)
	rows = append(rows, "     "+StyleCyan.Bold(true).Render(cmd))

	rows = append(rows, "")

	// noVNC web panel.
	novncDot := Dot(novncUp, "")
	rows = append(rows, "   "+StyleYellow.Render(fmt.Sprintf("%-17s", "noVNC Web Panel"))+"  "+novncDot)
	if novncUp {
		url := fmt.Sprintf("http://%s:6080/vnc.html", ip)
		rows = append(rows, "     "+StyleCyan.Bold(true).Render(url))
	} else {
		rows = append(rows, "     "+StyleRed.Render("(nginx/noVNC not running)"))
	}

	// Push separator and toggle to bottom.
	for len(rows) < height-3 {
		rows = append(rows, "")
	}

	rows = append(rows, StyleBorder.Render("├"+strings.Repeat("─", width)+"┤"))

	toggleLabel := boolStr(vncUp, "  Stop VNC  ", "  Start VNC  ")
	if cursor == 0 {
		rows = append(rows, StyleSelected.Render(toggleLabel))
	} else {
		rows = append(rows, StyleCyan.Bold(true).Render(toggleLabel))
	}

	return joinRows(rows, height, width)
}
