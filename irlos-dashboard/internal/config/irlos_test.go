// SPDX-License-Identifier: GPL-3.0-or-later
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIrlosRoundtrip(t *testing.T) {
	dir := t.TempDir()
	origPath := IrlosConfigPath
	// Redirect path to temp dir for the test.
	testPath := filepath.Join(dir, "config.json")
	// Patch via a helper that changes the package-level constant.
	// Since Go constants can't be patched, we write directly and then read.

	want := IrlosConfig{
		StreamKey: "live_abc123",
		Platform:  "Kick",
		WifiSSID:  "HomeNet",
		SSHPubKey: "ssh-ed25519 AAAA...",
		Hostname:  "irlos-box",
	}

	data, err := marshalIrlos(want)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := readIrlosFrom(testPath)
	if err != nil {
		t.Fatal(err)
	}

	if got != want {
		t.Errorf("roundtrip mismatch:\n  got  %+v\n  want %+v", got, want)
	}
	_ = origPath
}

func TestIrlosAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "config.json")

	cfg := IrlosConfig{Hostname: "test-host", Platform: "Twitch"}
	if err := writeIrlosTo(path, cfg); err != nil {
		t.Fatal(err)
	}

	// Verify file exists and reads back correctly.
	got, err := readIrlosFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Hostname != cfg.Hostname || got.Platform != cfg.Platform {
		t.Errorf("got %+v, want %+v", got, cfg)
	}
}

func TestIrlosMissingFile(t *testing.T) {
	_, err := readIrlosFrom("/nonexistent/path/config.json")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}
