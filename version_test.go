package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestVersionFilesMatchAppVersion(t *testing.T) {
	var wails struct {
		Info struct {
			ProductVersion string `json:"productVersion"`
		} `json:"info"`
	}
	readJSONFile(t, "wails.json", &wails)
	if wails.Info.ProductVersion != AppVersion {
		t.Errorf("wails.json productVersion = %q, want %q", wails.Info.ProductVersion, AppVersion)
	}

	var pkg struct {
		Version string `json:"version"`
	}
	readJSONFile(t, filepath.Join("frontend", "package.json"), &pkg)
	if pkg.Version != AppVersion {
		t.Errorf("frontend/package.json version = %q, want %q", pkg.Version, AppVersion)
	}
}

func readJSONFile(t *testing.T, path string, v interface{}) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
}
