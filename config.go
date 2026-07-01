package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// appConfig holds persisted user preferences from config.json.
type appConfig struct {
	WindowsPaths  []string `json:"windows_paths,omitempty"`
	MacosPaths    []string `json:"macos_paths,omitempty"`
	LastOutputDir string   `json:"last_output_dir,omitempty"`
}

// configPath returns the best path for config.json (next to exe or cwd).
func configPath() string {
	if exe, err := os.Executable(); err == nil {
		p := filepath.Join(filepath.Dir(exe), "config.json")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return filepath.Join(".", "config.json")
}

// loadConfig reads config.json (used only for custom WoW paths).
func loadConfig() (appConfig, error) {
	var cfg appConfig
	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg, err
	}
	err = json.Unmarshal(data, &cfg)
	return cfg, err
}

// saveConfig writes config.json only if it already exists.
func saveConfig(cfg appConfig) error {
	p := configPath()
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return nil
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

// loadLastOutputDir reads the last used output directory from OS-native storage.
// On Windows this uses the registry; on macOS it uses UserDefaults.
// Falls back to the default WoWBackups path if nothing is stored.
func loadLastOutputDir() string {
	return loadLastOutputDirOS()
}

// saveLastOutputDir persists the last used output directory to OS-native storage.
func saveLastOutputDir(dir string) {
	saveLastOutputDirOS(dir)
}
