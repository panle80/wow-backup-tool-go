//go:build windows

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// ---------------------------------------------------------------------------
// Windows registry detection
// ---------------------------------------------------------------------------

func (d *WoWDetector) detectWindowsRegistry() []WoWInstallation {
	var result []WoWInstallation

	// ---- Legacy Blizzard registry keys ----
	legacyKeys := []string{
		`SOFTWARE\Blizzard Entertainment\World of Warcraft`,
		`SOFTWARE\WOW6432Node\Blizzard Entertainment\World of Warcraft`,
	}
	for _, key := range legacyKeys {
		k, err := registry.OpenKey(registry.LOCAL_MACHINE, key, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		installPath, _, err := k.GetStringValue("InstallPath")
		k.Close()
		if err == nil && dirExists(installPath) {
			found := d.detectFromPaths([]string{installPath})
			if len(found) > 0 {
				logDetect("Found WoW via legacy registry: %s", installPath)
			}
			result = append(result, found...)
		}
	}

	// ---- Uninstall registry (newer Blizzard installs) ----
	uninstallBases := []string{
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
		`SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`,
	}
	for _, base := range uninstallBases {
		parent, err := registry.OpenKey(registry.LOCAL_MACHINE, base, registry.ENUMERATE_SUB_KEYS)
		if err != nil {
			continue
		}
		subNames, err := parent.ReadSubKeyNames(0)
		parent.Close()
		if err != nil {
			continue
		}

		for _, sub := range subNames {
			fullKey := base + `\` + sub
			sk, err := registry.OpenKey(registry.LOCAL_MACHINE, fullKey, registry.QUERY_VALUE)
			if err != nil {
				continue
			}

			dn, _, _ := sk.GetStringValue("DisplayName")
			if !strings.Contains(strings.ToLower(dn), "world of warcraft") {
				sk.Close()
				continue
			}

			il, _, err := sk.GetStringValue("InstallLocation")
			sk.Close()
			if err == nil && dirExists(il) {
				found := d.detectFromPaths([]string{il})
				if len(found) > 0 {
					logDetect("Found WoW via Uninstall registry: %s (%s)", il, dn)
				}
				result = append(result, found...)
			}
		}
	}

	return result
}

// ---------------------------------------------------------------------------
// Battle.net detection
// ---------------------------------------------------------------------------

func (d *WoWDetector) detectBattleNet() []WoWInstallation {
	var result []WoWInstallation

	// ---- Method 1: Battle.net.config JSON (current format) ----
	appdata := os.Getenv("APPDATA")
	if appdata != "" {
		cfgPath := filepath.Join(appdata, "Battle.net", "Battle.net.config")
		if data, err := os.ReadFile(cfgPath); err == nil {
			var cfg struct {
				Client struct {
					Install struct {
						DefaultInstallPath string `json:"DefaultInstallPath"`
					} `json:"Install"`
				} `json:"Client"`
			}
			if json.Unmarshal(data, &cfg) == nil && cfg.Client.Install.DefaultInstallPath != "" {
				wowPath := filepath.Join(cfg.Client.Install.DefaultInstallPath, "World of Warcraft")
				if dirExists(wowPath) {
					logDetect("Found WoW via Battle.net.config: %s", wowPath)
					result = append(result, d.detectFromPaths([]string{wowPath})...)
				}
			}
		}
	}

	// ---- Method 2: Legacy product.db (simple binary search) ----
	// The old Battle.net agent stored install paths in a SQLite product.db.
	// Since we want to avoid the CGO dependency of go-sqlite3, we do a
	// lightweight scan: read the file as raw bytes and look for path-like
	// strings containing "World of Warcraft". This is robust enough for the
	// simple key-value store that the legacy product.db was.
	programData := os.Getenv("PROGRAMDATA")
	if programData == "" {
		programData = `C:\ProgramData`
	}
	dbPath := filepath.Join(programData, "Battle.net", "Agent", "product.db")
	if data, err := os.ReadFile(dbPath); err == nil {
		// Scan for WoW install paths embedded in the SQLite file
		text := string(data)
		// Look for paths containing "World of Warcraft"
		for {
			idx := strings.Index(strings.ToLower(text), "world of warcraft")
			if idx < 0 {
				break
			}
			// Walk backward to find a plausible path start (drive letter or \\)
			start := idx
			for start > 0 && text[start-1] != 0 && text[start-1] != '/' && text[start-1] != '\\' {
				start--
			}
			// Find the end of this string
			end := idx
			for end < len(text) && text[end] != 0 && text[end] != '\n' && text[end] != '\r' {
				end++
			}
			candidate := strings.TrimSpace(text[start:end])
			candidate = strings.Trim(candidate, "\x00\t ")
			if dirExists(candidate) {
				found := d.detectFromPaths([]string{candidate})
				if len(found) > 0 {
					logDetect("Found WoW via product.db scan: %s", candidate)
				}
				result = append(result, found...)
			}
			text = text[end:]
		}
	}

	return result
}
