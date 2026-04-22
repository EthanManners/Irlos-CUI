// SPDX-License-Identifier: GPL-3.0-or-later
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const NoalbsConfigPath = "/home/irlos/.config/noalbs/config.json"

// NoalbsConfig holds the thresholds we expose in the TUI.
type NoalbsConfig struct {
	LowThresh     string // kbps, e.g. "2500"
	OfflineThresh string // kbps, e.g. "500"
}

// noalbsRoot is the full structure of the noalbs JSON file.
// We only touch switcher.thresholds; everything else is preserved.
type noalbsRoot struct {
	Switcher noalbsSwitcher         `json:"switcher"`
	Rest     map[string]interface{} `json:"-"`
}

type noalbsSwitcher struct {
	Thresholds noalbsThresholds       `json:"thresholds"`
	Rest       map[string]interface{} `json:"-"`
}

type noalbsThresholds struct {
	Low     int `json:"low"`
	Offline int `json:"offline"`
}

// ReadNoalbs reads the noalbs config file.
func ReadNoalbs() (NoalbsConfig, error) {
	defaults := NoalbsConfig{LowThresh: "2500", OfflineThresh: "500"}

	data, err := os.ReadFile(NoalbsConfigPath)
	if err != nil {
		return defaults, err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return defaults, err
	}

	switcher, ok := raw["switcher"]
	if !ok {
		return defaults, nil
	}
	var sw map[string]json.RawMessage
	if err := json.Unmarshal(switcher, &sw); err != nil {
		return defaults, nil
	}

	thresh, ok := sw["thresholds"]
	if !ok {
		return defaults, nil
	}
	var t noalbsThresholds
	if err := json.Unmarshal(thresh, &t); err != nil {
		return defaults, nil
	}

	return NoalbsConfig{
		LowThresh:     fmt.Sprintf("%d", t.Low),
		OfflineThresh: fmt.Sprintf("%d", t.Offline),
	}, nil
}

// WriteNoalbs merges updated thresholds into the existing noalbs config and
// writes atomically. If the file doesn't exist, a minimal skeleton is created.
func WriteNoalbs(cfg NoalbsConfig) error {
	// Parse ints with fallback.
	low := parseIntDefault(cfg.LowThresh, 2500)
	offline := parseIntDefault(cfg.OfflineThresh, 500)

	// Load existing document to preserve unknown keys.
	var doc map[string]json.RawMessage
	if data, err := os.ReadFile(NoalbsConfigPath); err == nil {
		_ = json.Unmarshal(data, &doc)
	}
	if doc == nil {
		doc = make(map[string]json.RawMessage)
	}

	// Merge thresholds.
	var sw map[string]json.RawMessage
	if raw, ok := doc["switcher"]; ok {
		_ = json.Unmarshal(raw, &sw)
	}
	if sw == nil {
		sw = make(map[string]json.RawMessage)
	}

	threshBytes, _ := json.Marshal(noalbsThresholds{Low: low, Offline: offline})
	sw["thresholds"] = threshBytes

	swBytes, err := json.Marshal(sw)
	if err != nil {
		return err
	}
	doc["switcher"] = swBytes

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(NoalbsConfigPath), 0o755); err != nil {
		return err
	}
	return atomicWrite(NoalbsConfigPath, out, 0o644)
}

func parseIntDefault(s string, def int) int {
	var v int
	if _, err := fmt.Sscanf(s, "%d", &v); err == nil {
		return v
	}
	return def
}
