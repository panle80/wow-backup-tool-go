package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ---------------------------------------------------------------------------
// Data types
// ---------------------------------------------------------------------------

// WoWInstallation represents a single WoW installation.
type WoWInstallation struct {
	Path        string `json:"path"`
	Version     string `json:"version"`
	DisplayName string `json:"displayName"`
}

// ---------------------------------------------------------------------------
// Version folder → display name mapping
// ---------------------------------------------------------------------------

var versionNames = map[string]string{
	"_retail_":        "正式服",
	"_classic_":       "经典怀旧服",
	"_classic_era_":   "经典旧世",
	"_classic_titan_": "大灾变怀旧服",
	"_forever_":       "无限服",
	"_classic_ptr_":   "怀旧服 PTR",
	"_ptr_":           "PTR",
	"_beta_":          "Beta",
}

// ---------------------------------------------------------------------------
// Default search paths
// ---------------------------------------------------------------------------

var defaultWindowsPaths = []string{
	`C:\Program Files (x86)\World of Warcraft`,
	`C:\Program Files\World of Warcraft`,
	`C:\Games\World of Warcraft`,
	`D:\World of Warcraft`,
	`D:\Games\World of Warcraft`,
	`E:\World of Warcraft`,
	`E:\Games\World of Warcraft`,
}

var defaultMacosPaths = []string{
	"/Applications/World of Warcraft",
	expandUser("~/Applications/World of Warcraft"),
	expandUser("~/Games/World of Warcraft"),
}

// ---------------------------------------------------------------------------
// WoWDetector
// ---------------------------------------------------------------------------

// WoWDetector detects World of Warcraft installations.
type WoWDetector struct {
	windowsPaths []string
	macosPaths   []string
}

// NewWoWDetector creates a detector, loading custom paths from config.json if present.
func NewWoWDetector() *WoWDetector {
	d := &WoWDetector{
		windowsPaths: defaultWindowsPaths,
		macosPaths:   defaultMacosPaths,
	}

	// Load custom paths from config.json (next to the executable or cwd)
	configNames := []string{"config.json"}
	if exe, err := os.Executable(); err == nil {
		configNames = append(configNames, filepath.Join(filepath.Dir(exe), "config.json"))
	}
	configNames = append(configNames, filepath.Join(".", "config.json"))

	for _, cfgPath := range configNames {
		if data, err := os.ReadFile(cfgPath); err == nil {
			var cfg struct {
				WindowsPaths []string `json:"windows_paths"`
				MacosPaths   []string `json:"macos_paths"`
			}
			if json.Unmarshal(data, &cfg) == nil {
				if len(cfg.WindowsPaths) > 0 {
					d.windowsPaths = cfg.WindowsPaths
				}
				if len(cfg.MacosPaths) > 0 {
					d.macosPaths = cfg.MacosPaths
				}
				break
			}
		}
	}

	return d
}

// DetectAll returns all detected WoW installations.
func (d *WoWDetector) DetectAll() []WoWInstallation {
	var installations []WoWInstallation

	switch runtime.GOOS {
	case "windows":
		installations = append(installations, d.detectWindowsRegistry()...)
		installations = append(installations, d.detectBattleNet()...)
		installations = append(installations, d.detectFromPaths(d.windowsPaths)...)
	case "darwin":
		installations = append(installations, d.detectFromPaths(d.macosPaths)...)
	default:
		installations = append(installations, d.detectFromPaths(d.windowsPaths)...)
	}

	// Deduplicate by normalised path
	seen := make(map[string]bool)
	unique := make([]WoWInstallation, 0, len(installations))
	for _, inst := range installations {
		key := strings.ToLower(filepath.Clean(inst.Path))
		if !seen[key] {
			seen[key] = true
			unique = append(unique, inst)
		}
	}
	return unique
}

// DetectAllFromPath returns every WoW installation found within the given path.
func (d *WoWDetector) DetectAllFromPath(path string) []WoWInstallation {
	if !dirExists(path) {
		return []WoWInstallation{}
	}

	// Case 1: path itself is a version folder (e.g. _retail_). An explicit
	// selection is accepted even when the folder is empty — restoring into a
	// freshly installed game (no Interface/WTF yet) must stay possible.
	base := filepath.Base(path)
	if isVersionFolderName(base) {
		return []WoWInstallation{newVersionInstallation(path, base)}
	}

	// Case 2: path contains one or more version folders. Unknown folders are
	// only listed when they contain game data, so stray `_old_`/`_backup_`
	// folders cannot be auto-selected for a destructive restore.
	entries, err := os.ReadDir(path)
	if err != nil {
		return []WoWInstallation{}
	}
	var result []WoWInstallation
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		n := e.Name()
		if !isVersionFolderName(n) {
			continue
		}
		full := filepath.Join(path, n)
		if !isKnownOrHasGameData(n, full) {
			continue
		}
		result = append(result, newVersionInstallation(full, n))
	}
	if len(result) > 0 {
		return result
	}

	// Case 3: Interface/ exists directly in path
	if dirExists(filepath.Join(path, "Interface")) {
		return []WoWInstallation{{
			Path:        path,
			Version:     "",
			DisplayName: "WoW (" + filepath.Base(path) + ")",
		}}
	}

	return []WoWInstallation{}
}

func newVersionInstallation(path, version string) WoWInstallation {
	dn := versionNames[version]
	if dn == "" {
		dn = version
	}
	return WoWInstallation{Path: path, Version: version, DisplayName: dn}
}

// isVersionFolderName reports whether name looks like a WoW version folder
// such as _retail_ or _forever_.
func isVersionFolderName(name string) bool {
	return len(name) > 2 && strings.HasPrefix(name, "_") && strings.HasSuffix(name, "_")
}

// hasGameData reports whether path contains at least one folder this tool
// backs up.
func hasGameData(path string) bool {
	for _, folder := range BackupFolders {
		if dirExists(filepath.Join(path, folder)) {
			return true
		}
	}
	return false
}

// isKnownOrHasGameData decides whether a version folder found by scanning a
// directory should be offered to the user. Known versions are always accepted
// (a freshly installed game has no data folders yet); unknown ones must
// contain game data so stray `_old_`/`_backup_` folders are ignored.
func isKnownOrHasGameData(version, path string) bool {
	if _, known := versionNames[version]; known {
		return true
	}
	return hasGameData(path)
}

// ValidateInstallation checks which key subdirectories exist.
func (d *WoWDetector) ValidateInstallation(path string) map[string]bool {
	hasInterface := dirExists(filepath.Join(path, "Interface"))
	hasWTF := dirExists(filepath.Join(path, "WTF"))
	return map[string]bool{
		"interface": hasInterface,
		"wtf":       hasWTF,
		"valid":     hasInterface && hasWTF,
	}
}

// ---------------------------------------------------------------------------
// Internal: scan version folders and no-version-folder directories
// ---------------------------------------------------------------------------

func (d *WoWDetector) detectFromPaths(paths []string) []WoWInstallation {
	var result []WoWInstallation
	for _, base := range paths {
		if !dirExists(base) {
			continue
		}
		found := false
		entries, err := os.ReadDir(base)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			n := e.Name()
			if !isVersionFolderName(n) {
				continue
			}
			full := filepath.Join(base, n)
			if !isKnownOrHasGameData(n, full) {
				continue
			}
			result = append(result, newVersionInstallation(full, n))
			found = true
		}
		// No version folder but Interface/ directly present
		if !found && dirExists(filepath.Join(base, "Interface")) {
			result = append(result, WoWInstallation{
				Path:        base,
				Version:     "",
				DisplayName: "WoW (" + filepath.Base(base) + ")",
			})
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func expandUser(path string) string {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[1:])
		}
	}
	return path
}

// logDetect is a tiny printf helper shared across files.
func logDetect(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[detector] "+format+"\n", args...)
}
