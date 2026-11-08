package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVersionNamesIncludesForever(t *testing.T) {
	if got := versionNames["_forever_"]; got != "无限服" {
		t.Fatalf("versionNames[_forever_] = %q, want %q", got, "无限服")
	}
}

func TestDetectFromPathsFindsForever(t *testing.T) {
	base := t.TempDir()
	forever := filepath.Join(base, "_forever_")
	for _, sub := range []string{"Interface", "WTF", "Fonts"} {
		if err := os.MkdirAll(filepath.Join(forever, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	d := &WoWDetector{}
	found := d.detectFromPaths([]string{base})
	if len(found) != 1 {
		t.Fatalf("got %d installations, want 1: %+v", len(found), found)
	}

	inst := found[0]
	if inst.Path != forever {
		t.Errorf("Path = %q, want %q", inst.Path, forever)
	}
	if inst.Version != "_forever_" {
		t.Errorf("Version = %q, want %q", inst.Version, "_forever_")
	}
	if inst.DisplayName != "无限服" {
		t.Errorf("DisplayName = %q, want %q", inst.DisplayName, "无限服")
	}
}

func TestDetectAllFromPathForeverFolder(t *testing.T) {
	forever := filepath.Join(t.TempDir(), "_forever_")
	if err := os.MkdirAll(filepath.Join(forever, "Interface"), 0o755); err != nil {
		t.Fatal(err)
	}

	d := &WoWDetector{}
	found := d.DetectAllFromPath(forever)
	if len(found) != 1 {
		t.Fatalf("got %d installations, want 1", len(found))
	}
	if found[0].DisplayName != "无限服" {
		t.Errorf("DisplayName = %q, want %q", found[0].DisplayName, "无限服")
	}
}

func TestDetectAllFromPathListsEveryVersion(t *testing.T) {
	base := t.TempDir()
	for _, version := range []string{"_classic_", "_forever_"} {
		if err := os.MkdirAll(filepath.Join(base, version, "Interface"), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	d := &WoWDetector{}
	found := d.DetectAllFromPath(base)
	if len(found) != 2 {
		t.Fatalf("got %d installations, want 2: %+v", len(found), found)
	}

	names := make(map[string]string, len(found))
	for _, inst := range found {
		names[inst.Version] = inst.DisplayName
	}
	if names["_forever_"] != "无限服" {
		t.Errorf("forever display name = %q, want %q", names["_forever_"], "无限服")
	}
}

func TestDetectFromPathsIgnoresStrayVersionFolder(t *testing.T) {
	base := t.TempDir()
	// _old_ matches the folder naming pattern but has no game data.
	if err := os.MkdirAll(filepath.Join(base, "_old_"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(base, "_retail_", "Interface"), 0o755); err != nil {
		t.Fatal(err)
	}

	d := &WoWDetector{}
	found := d.detectFromPaths([]string{base})
	if len(found) != 1 {
		t.Fatalf("got %d installations, want 1: %+v", len(found), found)
	}
	if found[0].Version != "_retail_" {
		t.Errorf("Version = %q, want %q", found[0].Version, "_retail_")
	}
}

func TestDetectFromPathsFindsUnknownVersionWithGameData(t *testing.T) {
	base := t.TempDir()
	// A future version missing from versionNames must still be detected.
	future := filepath.Join(base, "_midnight_")
	if err := os.MkdirAll(filepath.Join(future, "WTF"), 0o755); err != nil {
		t.Fatal(err)
	}

	d := &WoWDetector{}
	found := d.detectFromPaths([]string{base})
	if len(found) != 1 {
		t.Fatalf("got %d installations, want 1: %+v", len(found), found)
	}
	if found[0].Version != "_midnight_" {
		t.Errorf("Version = %q, want %q", found[0].Version, "_midnight_")
	}
	if found[0].DisplayName != "_midnight_" {
		t.Errorf("DisplayName = %q, want raw folder name", found[0].DisplayName)
	}
}

func TestDetectAllFromPathKeepsFreshKnownVersion(t *testing.T) {
	base := t.TempDir()
	// A freshly installed known version has no Interface/WTF/Fonts yet.
	if err := os.MkdirAll(filepath.Join(base, "_retail_"), 0o755); err != nil {
		t.Fatal(err)
	}

	d := &WoWDetector{}
	found := d.DetectAllFromPath(base)
	if len(found) != 1 || found[0].Version != "_retail_" {
		t.Fatalf("got %+v, want _retail_", found)
	}
}

func TestDetectAllFromPathIgnoresStrayUnknownFolder(t *testing.T) {
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, "_backup_"), 0o755); err != nil {
		t.Fatal(err)
	}

	d := &WoWDetector{}
	if found := d.DetectAllFromPath(base); len(found) != 0 {
		t.Fatalf("got %+v, want none", found)
	}
}

func TestDetectAllFromPathReturnsEmptySlice(t *testing.T) {
	d := &WoWDetector{}
	found := d.DetectAllFromPath(filepath.Join(t.TempDir(), "missing"))
	if found == nil {
		t.Fatal("DetectAllFromPath returned nil, want empty slice")
	}
	if len(found) != 0 {
		t.Fatalf("got %+v, want empty slice", found)
	}
}

func TestDetectAllFromPathAcceptsExplicitEmptyVersionFolder(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "_midnight_")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	d := &WoWDetector{}
	found := d.DetectAllFromPath(dir)
	if len(found) != 1 || found[0].Version != "_midnight_" {
		t.Fatalf("got %+v, want _midnight_", found)
	}
}

func TestDetectFromPathsMultipleVersions(t *testing.T) {
	base := t.TempDir()
	for _, version := range []string{"_retail_", "_classic_", "_forever_"} {
		if err := os.MkdirAll(filepath.Join(base, version, "Interface"), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	d := &WoWDetector{}
	found := d.detectFromPaths([]string{base})
	if len(found) != 3 {
		t.Fatalf("got %d installations, want 3: %+v", len(found), found)
	}

	names := make(map[string]string, len(found))
	for _, inst := range found {
		names[inst.Version] = inst.DisplayName
	}
	if names["_forever_"] != "无限服" {
		t.Errorf("forever display name = %q, want %q", names["_forever_"], "无限服")
	}
}
