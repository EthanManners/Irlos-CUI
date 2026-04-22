// SPDX-License-Identifier: GPL-3.0-or-later
package ui

import (
	"fmt"
	"strings"
)

// ConfigFieldKind describes how the field is edited.
type ConfigFieldKind int

const (
	ConfigKindInput  ConfigFieldKind = iota
	ConfigKindSelect                 // opens a select popup
)

// ConfigField is one editable row in the Config tab.
type ConfigField struct {
	Label  string
	Value  string
	Kind   ConfigFieldKind
	Masked bool
	Opts   []string // for select kind
}

// ConfigFields builds the ordered field list, applying staged values.
func ConfigFields(
	streamKey, platform, wifiSSID, sshPubKey, hostname string,
	staged map[string]string,
) []ConfigField {
	resolve := func(key, fallback string) string {
		if v, ok := staged[key]; ok {
			return v
		}
		return fallback
	}
	return []ConfigField{
		{"Stream Key", resolve("Stream Key", streamKey), ConfigKindInput, true, nil},
		{"Platform", resolve("Platform", platform), ConfigKindSelect, false, []string{"Kick", "Twitch", "YouTube", "Custom RTMP"}},
		{"WiFi SSID", resolve("WiFi SSID", wifiSSID), ConfigKindInput, false, nil},
		{"SSH PubKey", resolve("SSH PubKey", sshPubKey), ConfigKindInput, false, nil},
		{"Hostname", resolve("Hostname", hostname), ConfigKindInput, false, nil},
	}
}

// RenderConfig renders the Config tab inner content.
func RenderConfig(width, height, cursor int, fields []ConfigField, staged map[string]string) string {
	var rows []string
	rows = append(rows, "") // top padding

	for i, f := range fields {
		sel := cursor == i
		stgd := staged[f.Label] != ""

		displayVal := f.Value
		if f.Masked && f.Value != "" {
			n := len(f.Value)
			if n > 20 {
				n = 20
			}
			displayVal = strings.Repeat("●", n) + fmt.Sprintf("  (%d chars)", len(f.Value))
		}

		var lStr, vStr string
		if sel {
			lStr = StyleSelected.Render(fmt.Sprintf("%-16s", f.Label))
			vStr = StyleSelected.Render(fmt.Sprintf("%-36s", displayVal))
		} else {
			lStr = StyleYellow.Render(fmt.Sprintf("%-16s", f.Label))
			vStr = StyleDefault.Render(fmt.Sprintf("%-36s", displayVal))
		}

		row := "   " + lStr + vStr
		if stgd {
			row += "  " + StyleStaged.Render("[staged]")
		}
		rows = append(rows, row)
	}

	rows = append(rows, StyleBorder.Render("├"+strings.Repeat("─", width)+"┤"))

	sel := cursor == len(fields)
	applyStr := "  ► Apply Config Changes  "
	if sel {
		rows = append(rows, StyleSelected.Render(applyStr))
	} else {
		rows = append(rows, StyleGreen.Bold(true).Render(applyStr))
	}

	return joinRows(rows, height, width)
}
