// SPDX-License-Identifier: GPL-3.0-or-later
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const IrlosConfigPath = "/etc/irlos/config.json"

// IrlosConfig is the schema for /etc/irlos/config.json.
type IrlosConfig struct {
	StreamKey string `json:"stream_key"`
	Platform  string `json:"platform"`
	WifiSSID  string `json:"wifi_ssid"`
	SSHPubKey string `json:"ssh_pubkey"`
	Hostname  string `json:"hostname"`
}

// ReadIrlos reads the irlos config file. Missing fields are left at their
// zero values; callers should supply defaults in the UI layer.
func ReadIrlos() (IrlosConfig, error) {
	f, err := os.Open(IrlosConfigPath)
	if err != nil {
		return IrlosConfig{}, err
	}
	defer f.Close()

	var cfg IrlosConfig
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return IrlosConfig{}, err
	}
	return cfg, nil
}

// WriteIrlos writes cfg to IrlosConfigPath atomically.
func WriteIrlos(cfg IrlosConfig) error {
	if err := os.MkdirAll(filepath.Dir(IrlosConfigPath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(IrlosConfigPath, data, 0o644)
}
