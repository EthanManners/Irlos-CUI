// SPDX-License-Identifier: GPL-3.0-or-later
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOBSRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "basic.ini")

	want := OBSConfig{Resolution: "1280x720", FPS: "30", Bitrate: "4500"}
	if err := writeOBSTo(path, want); err != nil {
		t.Fatal(err)
	}

	got, err := readOBSFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("roundtrip mismatch: got %+v, want %+v", got, want)
	}
}

func TestOBSDefaults(t *testing.T) {
	_, err := readOBSFrom("/nonexistent/basic.ini")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestOBSPreservesUnknownKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "basic.ini")

	// Write a file with extra keys.
	initial := "[Video]\nBaseCX=1920\nBaseCY=1080\nFPSNumerator=60\nSomeOtherKey=preserved\n[SimpleOutput]\nVBitrate=6000\n"
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := readOBSFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Bitrate = "8000"
	if err := writeOBSTo(path, cfg); err != nil {
		t.Fatal(err)
	}

	// Read back and verify bitrate changed.
	got, err := readOBSFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Bitrate != "8000" {
		t.Errorf("bitrate: got %q, want %q", got.Bitrate, "8000")
	}
}
