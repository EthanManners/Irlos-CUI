// SPDX-License-Identifier: GPL-3.0-or-later
package config

import (
	"gopkg.in/ini.v1"
	"os"
	"path/filepath"
	"strings"
)

func readOBSFrom(path string) (OBSConfig, error) {
	cfg := OBSDefaults()
	f, err := ini.LoadSources(ini.LoadOptions{Insensitive: true}, path)
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
		cfg.Resolution = w + "x" + h
	}
	return cfg, nil
}

func writeOBSTo(path string, cfg OBSConfig) error {
	f, err := ini.LoadSources(ini.LoadOptions{Insensitive: false}, path)
	if err != nil {
		f = ini.Empty()
	}
	parts := strings.SplitN(cfg.Resolution, "x", 2)
	if len(parts) != 2 {
		parts = []string{"1920", "1080"}
	}
	w, h := parts[0], parts[1]

	for _, secName := range []string{"Video"} {
		s, serr := f.GetSection(secName)
		if serr != nil {
			s, _ = f.NewSection(secName)
		}
		s.Key("BaseCX").SetValue(w)
		s.Key("BaseCY").SetValue(h)
		s.Key("OutputCX").SetValue(w)
		s.Key("OutputCY").SetValue(h)
		s.Key("FPSNumerator").SetValue(cfg.FPS)
	}
	for _, secName := range []string{"SimpleOutput"} {
		s, serr := f.GetSection(secName)
		if serr != nil {
			s, _ = f.NewSection(secName)
		}
		s.Key("VBitrate").SetValue(cfg.Bitrate)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".obs-test-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := f.WriteTo(tmp); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	tmp.Close()
	return os.Rename(name, path)
}
