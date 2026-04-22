// SPDX-License-Identifier: GPL-3.0-or-later
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNoalbsRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	want := NoalbsConfig{LowThresh: "3000", OfflineThresh: "400"}
	if err := writeNoalbsTo(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := readNoalbsFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("roundtrip mismatch: got %+v, want %+v", got, want)
	}
}

func TestNoalbsPreservesOtherKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	// Write a full noalbs config with extra fields.
	initial := map[string]interface{}{
		"switcher": map[string]interface{}{
			"thresholds": map[string]int{"low": 2500, "offline": 500},
			"switchBRB":  true,
		},
		"logLevel": "info",
	}
	data, _ := json.MarshalIndent(initial, "", "  ")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := writeNoalbsTo(path, NoalbsConfig{LowThresh: "1000", OfflineThresh: "200"}); err != nil {
		t.Fatal(err)
	}

	// Read raw JSON and verify logLevel still present.
	var doc map[string]json.RawMessage
	raw, _ := os.ReadFile(path)
	_ = json.Unmarshal(raw, &doc)
	if _, ok := doc["logLevel"]; !ok {
		t.Error("logLevel key was not preserved after write")
	}
}

func TestNoalbsMissingFile(t *testing.T) {
	_, err := readNoalbsFrom("/nonexistent/noalbs/config.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}
