// SPDX-License-Identifier: GPL-3.0-or-later
package ui

import (
	"fmt"
	"strings"
)

// OBSFieldKind identifies how a field is rendered and interacted with.
type OBSFieldKind int

const (
	OBSKindSelect OBSFieldKind = iota
	OBSKindInput
	OBSKindDisplay // read-only
)

// OBSField describes one row in the OBS tab.
type OBSField struct {
	Label string
	Value string
	Kind  OBSFieldKind
}

// OBSFields builds the ordered field list with staged values applied.
func OBSFields(resolution, fps, bitrate, platform string, staged map[string]string) []OBSField {
	resolve := func(key, fallback string) string {
		if v, ok := staged[key]; ok {
			return v
		}
		return fallback
	}
	return []OBSField{
		{"Resolution", resolve("Resolution", resolution), OBSKindSelect},
		{"FPS", resolve("FPS", fps), OBSKindSelect},
		{"Bitrate", resolve("Bitrate", bitrate), OBSKindInput},
		{"Encoder", "NVENC H.264", OBSKindDisplay},
		{"Platform", resolve("Platform", platform), OBSKindSelect},
	}
}

// RenderOBS renders the OBS tab inner content.
func RenderOBS(width, height, cursor int, fields []OBSField, staged map[string]string) string {
	var rows []string

	rows = append(rows, "") // top padding

	for i, f := range fields {
		sel := cursor == i
		stgd := staged[f.Label] != ""

		var lStr, vStr string
		switch {
		case f.Kind == OBSKindDisplay:
			lStr = StyleDim.Render(fmt.Sprintf("%-18s", f.Label))
			vStr = StyleDim.Render(fmt.Sprintf("%-22s", f.Value))
		case sel:
			lStr = StyleSelected.Render(fmt.Sprintf("%-18s", f.Label))
			vStr = StyleSelected.Render(fmt.Sprintf("%-22s", f.Value))
		default:
			lStr = StyleYellow.Render(fmt.Sprintf("%-18s", f.Label))
			vStr = StyleDefault.Render(fmt.Sprintf("%-22s", f.Value))
		}

		row := "   " + lStr + vStr
		if stgd {
			row += "  " + StyleStaged.Render("[staged]")
		}
		rows = append(rows, row)
	}

	rows = append(rows, StyleBorder.Render("├"+strings.Repeat("─", width)+"┤"))

	// Apply action.
	sel := cursor == len(fields)
	applyStr := "  ► Apply Changes & Restart OBS  "
	if sel {
		rows = append(rows, StyleSelected.Render(applyStr))
	} else {
		rows = append(rows, StyleGreen.Bold(true).Render(applyStr))
	}

	return joinRows(rows, height, width)
}
