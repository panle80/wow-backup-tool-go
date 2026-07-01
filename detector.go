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
	var unique []WoWInstallation
	for _, inst := range installations {
		key := strings.ToLower(filepath.Clean(inst.Path))
		if !seen[key] {
			seen[key] = true
			unique = append(unique, inst)
		}
	}
	return unique
}

// DetectFromPath tries to locate a WoW installation within the given path.
func (d *WoWDetector) DetectFromPath(path string) *WoWInstallation {
	if !dirExists(path) {
		return nil
	}

	base := filepath.Base(path)

	// Case 1: path itself is a version folder (e.g. _retail_)
	if strings.HasPrefix(base, "_") && strings.HasSuffix(base, "_") {
		dn := versionNames[base]
		if dn == "" {
			dn = base
		}
		return &WoWInstallation{Path: path, Version: base, DisplayName: dn}
	}

	// Case 2: path contains version folders — pick the first one
	entries, err := os.ReadDir(path)
	if err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			n := e.Name()
			if strings.HasPrefix(n, "_") && strings.HasSuffix(n, "_") {
				dn := versionNames[n]
				if dn == "" {
					dn = n
				}
				return &WoWInstallation{
					Path:        filepath.Join(path, n),
					Version:     n,
					DisplayName: dn,
				}
			}
		}
	}

	// Case 3: Interface/ exists directly in path
	if dirExists(filepath.Join(path, "Interface")) {
		return &WoWInstallation{
			Path:        path,
			Version:     "",
			DisplayName: "WoW (" + filepath.Base(path) + ")",
		}
	}

	return nil
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
			if dn, ok := versionNames[n]; ok {
				result = append(result, WoWInstallation{
					Path:        filepath.Join(base, n),
					Version:     n,
					DisplayName: dn,
				})
				found = true
			}
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

// fileExists reports whether a regular file exists at path.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// logDetect is a tiny printf helper shared across files.
func logDetect(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[detector] "+format+"\n", args...)
}
