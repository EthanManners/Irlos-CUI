// SPDX-License-Identifier: GPL-3.0-or-later
package model

import "github.com/ethanmanners/irlos-dashboard/internal/config"

// PolledState holds all data read from the system on each 2-second tick.
// Fields are zero-valued (not error-valued) so the UI always has something
// safe to render.
type PolledState struct {
	// Resource utilisation
	CPU      float64 // 0–100
	RAMUsed  uint64  // MiB
	RAMTotal uint64  // MiB
	GPUUtil  int     // 0–100; -1 = NVML unavailable
	GPUTemp  int     // °C; -1 = NVML unavailable

	// Service active-state booleans
	StreamLive bool
	OBSUp      bool
	NoalbsUp   bool
	SLSUp      bool
	VNCUp      bool
	NovncUp    bool
	NginxUp    bool

	// Network / stream info
	IP      string
	Scene   string // current OBS scene name
	RecvBPS string // SLS recv bitrate label
	SendBPS string // SLS send bitrate label
	Uptime  string // stream uptime hh:mm:ss

	// Config snapshots (read from disk each poll)
	IrlosCfg  config.IrlosConfig
	OBSCfg    config.OBSConfig
	NoalbsCfg config.NoalbsConfig
}
