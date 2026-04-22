// SPDX-License-Identifier: GPL-3.0-or-later
// Internal helpers exposed only to the test files in this package.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func marshalIrlos(cfg IrlosConfig) ([]byte, error) {
	return json.MarshalIndent(cfg, "", "  ")
}

func readIrlosFrom(path string) (IrlosConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return IrlosConfig{}, err
	}
	defer f.Close()
	var cfg IrlosConfig
	return cfg, json.NewDecoder(f).Decode(&cfg)
}

func writeIrlosTo(path string, cfg IrlosConfig) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(path, data, 0o644)
}
