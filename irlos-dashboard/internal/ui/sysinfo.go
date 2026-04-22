// SPDX-License-Identifier: GPL-3.0-or-later
package ui

import (
	"fmt"
	"strings"
)

// ServiceStatus is a (name, active) pair for the service row.
type ServiceStatus struct {
	Name   string
	Active bool
}

// RenderSysinfo renders the System info box (8 lines including borders).
func RenderSysinfo(
	width int,
	cpu float64,
	ramUsed, ramTotal uint64,
	gpuUtil, gpuTemp int,
	services []ServiceStatus,
) string {
	const boxHeight = 8
	boxWidth := min(width-4, 68)
	if boxWidth < 20 {
		boxWidth = 20
	}
	leftPad := (width - boxWidth) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	pad := strings.Repeat(" ", leftPad)

	inner := boxWidth - 2
	barWidth := 24

	lines := make([]string, boxHeight)

	// Top border.
	lines[0] = pad + StyleBorder.Render("┌"+centerTitle("System", boxWidth-2)+"┐")

	// CPU bar.
	lines[1] = pad + "│" + padRight(Bar(cpu/100, barWidth, "CPU", fmt.Sprintf("%5.1f%%", cpu)), inner) + "│"

	// RAM bar.
	ramPct := 0.0
	if ramTotal > 0 {
		ramPct = float64(ramUsed) / float64(ramTotal)
	}
	lines[2] = pad + "│" + padRight(Bar(ramPct, barWidth, "RAM",
		fmt.Sprintf("%dMB / %dMB", ramUsed, ramTotal)), inner) + "│"

	// GPU util bar (or N/A).
	if gpuUtil >= 0 {
		lines[3] = pad + "│" + padRight(Bar(float64(gpuUtil)/100, barWidth, "GPU Util",
			fmt.Sprintf("%3d%%", gpuUtil)), inner) + "│"
	} else {
		lines[3] = pad + "│" + padRight(StyleDim.Render("GPU Util     N/A (no NVML)"), inner) + "│"
	}

	// GPU temp bar (or N/A).
	if gpuTemp >= 0 {
		lines[4] = pad + "│" + padRight(Bar(float64(gpuTemp)/100, barWidth, "GPU Temp",
			fmt.Sprintf("%3d°C", gpuTemp)), inner) + "│"
	} else {
		lines[4] = pad + "│" + padRight(StyleDim.Render("GPU Temp     N/A"), inner) + "│"
	}

	// Blank separator row.
	lines[5] = pad + "│" + strings.Repeat(" ", inner) + "│"

	// Service dots row.
	var dotRow strings.Builder
	for _, svc := range services {
		seg := Dot(svc.Active, "") + " " + StyleDefault.Render(svc.Name)
		dotRow.WriteString("  ")
		dotRow.WriteString(seg)
	}
	dotStr := dotRow.String()
	if strings.TrimSpace(dotStr) == "" {
		dotStr = ""
	}
	lines[6] = pad + "│" + padRight(dotStr, inner) + "│"

	// Bottom border.
	lines[7] = pad + StyleBorder.Render("└"+strings.Repeat("─", boxWidth-2)+"┘")

	return strings.Join(lines, "\n")
}

func centerTitle(title string, width int) string {
	t := " " + title + " "
	pad := (width - len(t)) / 2
	if pad < 0 {
		pad = 0
	}
	left := strings.Repeat("─", pad)
	right := strings.Repeat("─", width-pad-len(t))
	return StyleBorder.Render(left) + StyleBorder.Bold(true).Render(t) + StyleBorder.Render(right)
}

// padRight pads or truncates s to exactly width visible characters.
func padRight(s string, width int) string {
	sw := VisibleWidth(s)
	if sw < width {
		return s + strings.Repeat(" ", width-sw)
	}
	return s
}

// VisibleWidth returns the number of visible (non-ANSI-escape) characters in s.
// This is exported for use by the model package.
func VisibleWidth(s string) int {
	w := 0
	inEsc := false
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		w++
	}
	return w
}
