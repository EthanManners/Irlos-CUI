// SPDX-License-Identifier: GPL-3.0-or-later
package config

import (
	"encoding/json"
	"fmt"
	"os"
)

func readNoalbsFrom(path string) (NoalbsConfig, error) {
	defaults := NoalbsConfig{LowThresh: "2500", OfflineThresh: "500"}
	data, err := os.ReadFile(path)
	if err != nil {
		return defaults, err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return defaults, err
	}
	sw, ok := raw["switcher"]
	if !ok {
		return defaults, nil
	}
	var swMap map[string]json.RawMessage
	if err := json.Unmarshal(sw, &swMap); err != nil {
		return defaults, nil
	}
	thresh, ok := swMap["thresholds"]
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

func writeNoalbsTo(path string, cfg NoalbsConfig) error {
	low := parseIntDefault(cfg.LowThresh, 2500)
	offline := parseIntDefault(cfg.OfflineThresh, 500)

	var doc map[string]json.RawMessage
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &doc)
	}
	if doc == nil {
		doc = make(map[string]json.RawMessage)
	}

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
	return atomicWrite(path, out, 0o644)
}
