// SPDX-License-Identifier: GPL-3.0-or-later
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/ini.v1"
)

const OBSIniPath = "/home/irlos/.config/obs-studio/basic/profiles/irlos/basic.ini"

// OBSConfig holds the fields we care about from OBS's basic.ini.
type OBSConfig struct {
	Resolution string // e.g. "1920x1080"
	FPS        string // e.g. "60"
	Bitrate    string // VBitrate in kbps
}

// Defaults returns an OBSConfig populated with sane defaults.
func OBSDefaults() OBSConfig {
	return OBSConfig{Resolution: "1920x1080", FPS: "60", Bitrate: "6000"}
}

// ReadOBS parses OBSIniPath. Missing or corrupt files yield OBSDefaults().
func ReadOBS() (OBSConfig, error) {
	cfg := OBSDefaults()

	f, err := ini.LoadSources(ini.LoadOptions{Insensitive: true, AllowShadows: false}, OBSIniPath)
	if err != nil {
		return cfg, err
	}

	var w, h string
	for _, sec := range f.Sections() {
		for _, key := range sec.Keys() {
			switch strings.ToLower(key.Name()) {
			case "basecx":
				w = key.Value()
			case "basecy":
				h = key.Value()
			case "fpsnumerator":
				cfg.FPS = key.Value()
			case "vbitrate", "bitrate":
				cfg.Bitrate = key.Value()
			}
		}
	}
	if w != "" && h != "" {
		cfg.Resolution = fmt.Sprintf("%sx%s", w, h)
	}
	return cfg, nil
}

// WriteOBS writes updated values back to OBSIniPath atomically.
// Only Resolution, FPS, and Bitrate are touched; all other keys are preserved.
func WriteOBS(cfg OBSConfig) error {
	// Load existing file (or start empty).
	f, err := ini.LoadSources(ini.LoadOptions{Insensitive: false, AllowShadows: false}, OBSIniPath)
	if err != nil {
		f = ini.Empty()
	}

	parts := strings.SplitN(cfg.Resolution, "x", 2)
	if len(parts) != 2 {
		parts = []string{"1920", "1080"}
	}
	w, h := parts[0], parts[1]

	for _, sec := range []string{"Video"} {
		s, err := f.GetSection(sec)
		if err != nil {
			s, _ = f.NewSection(sec)
		}
		s.Key("BaseCX").SetValue(w)
		s.Key("BaseCY").SetValue(h)
		s.Key("OutputCX").SetValue(w)
		s.Key("OutputCY").SetValue(h)
		s.Key("FPSNumerator").SetValue(cfg.FPS)
	}

	for _, sec := range []string{"SimpleOutput"} {
		s, err := f.GetSection(sec)
		if err != nil {
			s, _ = f.NewSection(sec)
		}
		s.Key("VBitrate").SetValue(cfg.Bitrate)
	}

	if err := os.MkdirAll(filepath.Dir(OBSIniPath), 0o755); err != nil {
		return err
	}

	// Write to temp file then rename.
	tmp, err := os.CreateTemp(filepath.Dir(OBSIniPath), ".obs-*.ini.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err := f.WriteTo(tmp); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	tmp.Close()

	return os.Rename(tmpName, OBSIniPath)
}
