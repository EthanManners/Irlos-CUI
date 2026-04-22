// SPDX-License-Identifier: GPL-3.0-or-later
package ui

import (
	"fmt"
	"strings"
)

// NoalbsField is one editable row in the noalbs tab.
type NoalbsField struct {
	Key   string
	Label string
	Value string
}

// NoalbsFields builds the field list with staged values applied.
func NoalbsFields(low, offline string, staged map[string]string) []NoalbsField {
	resolve := func(key, fallback string) string {
		if v, ok := staged[key]; ok {
			return v
		}
		return fallback
	}
	return []NoalbsField{
		{"low", "Low Bitrate Thresh", resolve("low", low)},
		{"offline", "Offline Bitrate Thresh", resolve("offline", offline)},
	}
}

// RenderNoalbs renders the noalbs tab inner content.
func RenderNoalbs(width, height, cursor int, fields []NoalbsField, staged map[string]string) string {
	var rows []string
	rows = append(rows, "") // top padding

	for i, f := range fields {
		sel := cursor == i
		stgd := staged[f.Key] != ""

		var lStr, vStr string
		if sel {
			lStr = StyleSelected.Render(fmt.Sprintf("%-26s", f.Label))
			vStr = StyleSelected.Render(f.Value + " kbps")
		} else {
			lStr = StyleYellow.Render(fmt.Sprintf("%-26s", f.Label))
			vStr = StyleDefault.Render(f.Value + " kbps")
		}
		row := "   " + lStr + vStr
		if stgd {
			row += "  " + StyleStaged.Render("[staged]")
		}
		rows = append(rows, row)
	}

	rows = append(rows, "")
	rows = append(rows, "   "+StyleCyan.Bold(true).Render("Scene Mappings"))

	type mapping struct {
		state string
		scene string
		style interface{ Render(...string) string }
	}
	mappings := []mapping{
		{"Normal", "Live", StyleGreen},
		{"Low", "BRB", StyleYellow},
		{"Offline", "Offline", StyleRed},
	}
	for _, m := range mappings {
		rows = append(rows, fmt.Sprintf("     %s",
			m.style.Render(fmt.Sprintf("%-10s", m.state))+
				StyleDefault.Render("→  "+m.scene)))
	}

	rows = append(rows, StyleBorder.Render("├"+strings.Repeat("─", width)+"┤"))

	sel := cursor == len(fields)
	applyStr := "  ► Apply Changes & Restart noalbs  "
	if sel {
		rows = append(rows, StyleSelected.Render(applyStr))
	} else {
		rows = append(rows, StyleGreen.Bold(true).Render(applyStr))
	}

	return joinRows(rows, height, width)
}
