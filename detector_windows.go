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

	// ---- Method 2: Legacy product.db ----
	// Old Battle.net agent stored install paths in a SQLite database.
	// We avoid CGO by scanning raw bytes for absolute Windows paths
	// containing "World of Warcraft", validated before touching disk.
	programData := os.Getenv("PROGRAMDATA")
	if programData == "" {
		programData = `C:\ProgramData`
	}
	dbPath := filepath.Join(programData, "Battle.net", "Agent", "product.db")
	if data, err := os.ReadFile(dbPath); err == nil {
		lower := strings.ToLower(string(data))
		searchFrom := 0
		for {
			idx := strings.Index(lower[searchFrom:], "world of warcraft")
			if idx < 0 {
				break
			}
			idx += searchFrom
			searchFrom = idx + len("world of warcraft")

			// Backtrack to a drive letter (X:) or UNC (\\)
			start := idx
			for start > 1 {
				if (lower[start-1] == ':' && start >= 2 &&
					lower[start-2] >= 'a' && lower[start-2] <= 'z') ||
					(lower[start-1] == '\\' && start >= 2 &&
						lower[start-2] == '\\') {
					start -= 2
					break
				}
				if lower[start-1] == 0 || lower[start-1] == '\n' {
					break
				}
				start--
			}
			if start >= idx-2 {
				continue // no valid path prefix
			}

			// Forward to path end
			end := idx
			for end < len(lower) {
				b := lower[end]
				if b == 0 || b == '\n' || b == '\r' || b == '"' {
					break
				}
				end++
			}

			candidate := strings.TrimSpace(string(data[start:end]))
			candidate = strings.Trim(candidate, "\x00\t \"'")

			if len(candidate) < 10 || !dirExists(candidate) {
				continue
			}

			found := d.detectFromPaths([]string{candidate})
			if len(found) > 0 {
				logDetect("Found WoW via product.db: %s", candidate)
			}
			result = append(result, found...)
		}
	}

	return result
}
