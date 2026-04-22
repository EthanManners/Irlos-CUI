// SPDX-License-Identifier: GPL-3.0-or-later
package model

// Tab identifies which top-level panel is active.
type Tab int

const (
	TabStream Tab = iota
	TabOBS
	TabNoalbs
	TabVNC
	TabLogs
	TabConfig
)

// TabNames is the ordered display names for the tab bar.
var TabNames = []string{"Stream", "OBS", "noalbs", "VNC", "Logs", "Config"}

// TabCount is the number of tabs.
const TabCount = 6

func (t Tab) String() string {
	if int(t) < len(TabNames) {
		return TabNames[t]
	}
	return "Unknown"
}
