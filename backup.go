package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

// BackupFolders lists the folders that can be backed up / restored.
var BackupFolders = []string{"Interface", "WTF", "Fonts"}

// DefaultOutputDir is where backups are saved by default.
func DefaultOutputDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Documents", "WoWBackups")
}

// ---------------------------------------------------------------------------
// RestoreResult
// ---------------------------------------------------------------------------

// RestoreResult wraps the outcome of a restore operation.
type RestoreResult struct {
	Manifest   *BackupManifest `json:"manifest"`
	ConfigKept bool            `json:"configKept"`
}

// ---------------------------------------------------------------------------
// BackupManager
// ---------------------------------------------------------------------------

// BackupManager orchestrates detection, backup, and restore.
type BackupManager struct {
	detector *WoWDetector
	zipper   *ZipManager
}

// NewBackupManager creates a ready-to-use BackupManager.
func NewBackupManager() *BackupManager {
	return &BackupManager{
		detector: NewWoWDetector(),
		zipper:   NewZipManager(),
	}
}

// DetectInstallations returns all found WoW installations.
func (m *BackupManager) DetectInstallations() []WoWInstallation {
	return m.detector.DetectAll()
}

// DetectFromPath attempts to detect an installation at the given path.
func (m *BackupManager) DetectFromPath(path string) *WoWInstallation {
	return m.detector.DetectFromPath(path)
}

// ValidateInstallation checks which key folders exist under the given path.
func (m *BackupManager) ValidateInstallation(path string) map[string]bool {
	return m.detector.ValidateInstallation(path)
}

// CreateBackup builds a ZIP backup of the selected folders.
// progressFn receives (message, current, total).
func (m *BackupManager) CreateBackup(
	installation WoWInstallation,
	selectedFolders []string,
	outputDir string,
	progressFn func(string, int, int),
) (string, error) {
	// Build source path map: folder name → absolute path
	sourcePaths := make(map[string]string)
	for _, folder := range selectedFolders {
		actualPath := filepath.Join(installation.Path, folder)
		if dirExists(actualPath) {
			sourcePaths[folder] = actualPath
		} else {
			fmt.Fprintf(os.Stderr, "[backup] folder not found, skipping: %s\n", actualPath)
		}
	}
	if len(sourcePaths) == 0 {
		return "", fmt.Errorf("没有找到任何可备份的文件夹")
	}

	// Ensure output directory exists
	saveDir := outputDir
	if saveDir == "" {
		saveDir = DefaultOutputDir()
	}
	if err := os.MkdirAll(saveDir, 0o755); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}

	// Generate filename
	ts := time.Now().Format("20060102_150405")
	versionName := strings.Trim(installation.Version, "_")
	if versionName == "" {
		versionName = "WoW"
	}
	filename := fmt.Sprintf("WoWBackup_%s_%s.zip", versionName, ts)
	outputPath := filepath.Join(saveDir, filename)

	manifest := NewManifest(selectedFolders)

	fmt.Fprintf(os.Stderr, "[backup] creating: %s → %s\n", installation.DisplayName, outputPath)

	if err := m.zipper.CreateBackup(sourcePaths, outputPath, manifest, progressFn); err != nil {
		return "", fmt.Errorf("create backup: %w", err)
	}

	fmt.Fprintf(os.Stderr, "[backup] complete: %s\n", outputPath)
	return outputPath, nil
}

// RestoreBackup extracts a backup ZIP to an installation, mirror-restore style.
func (m *BackupManager) RestoreBackup(
	backupPath string,
	installation WoWInstallation,
	selectedFolders []string,
	keepLocalConfig bool,
	progressFn func(string, int, int),
) (*RestoreResult, error) {
	fmt.Fprintf(os.Stderr, "[backup] restoring: %s → %s (folders: %v)\n",
		backupPath, installation.Path, selectedFolders)

	folders := selectedFolders
	if len(folders) == 0 {
		folders = BackupFolders
	}

	// ---- Save local Config.wtf if requested ----
	var savedConfig []byte
	configKept := false
	if keepLocalConfig && contains(folders, "WTF") {
		configPath := filepath.Join(installation.Path, "WTF", "Config.wtf")
		if data, err := os.ReadFile(configPath); err == nil {
			savedConfig = data
			fmt.Fprintf(os.Stderr, "[backup] saved local Config.wtf (%d bytes)\n", len(data))
		}
	}

	// ---- Delete target folders (mirror restore) ----
	for _, folder := range folders {
		folderPath := filepath.Join(installation.Path, folder)
		if dirExists(folderPath) {
			if err := os.RemoveAll(folderPath); err != nil {
				return nil, fmt.Errorf("remove %s: %w", folderPath, err)
			}
			fmt.Fprintf(os.Stderr, "[backup] removed: %s\n", folderPath)
		}
	}

	// ---- Extract ZIP ----
	manifest, err := m.zipper.ExtractBackup(backupPath, installation.Path, folders, progressFn)
	if err != nil {
		return nil, fmt.Errorf("extract backup: %w", err)
	}

	// ---- Write back saved Config.wtf ----
	if savedConfig != nil {
		wtfDir := filepath.Join(installation.Path, "WTF")
		if err := os.MkdirAll(wtfDir, 0o755); err != nil {
			return nil, fmt.Errorf("mkdir WTF: %w", err)
		}
		configPath := filepath.Join(wtfDir, "Config.wtf")
		if err := os.WriteFile(configPath, savedConfig, 0o644); err != nil {
			return nil, fmt.Errorf("write Config.wtf: %w", err)
		}
		fmt.Fprintf(os.Stderr, "[backup] restored local Config.wtf\n")
		configKept = true
	}

	fmt.Fprintf(os.Stderr, "[backup] restore complete: %s\n", installation.DisplayName)
	return &RestoreResult{Manifest: manifest, ConfigKept: configKept}, nil
}

// GetBackupInfo returns metadata from a backup ZIP.
func (m *BackupManager) GetBackupInfo(backupPath string) (map[string]interface{}, error) {
	return m.zipper.GetBackupInfo(backupPath)
}

// ListBackups returns metadata for all backups in a directory.
func (m *BackupManager) ListBackups(directory string) ([]map[string]interface{}, error) {
	dir := directory
	if dir == "" {
		dir = DefaultOutputDir()
	}
	return m.zipper.ListBackups(dir)
}

// OpenFolder opens the given path in the platform file manager.
func (m *BackupManager) OpenFolder(path string) error {
	// Platform-specific open is handled in open_*.go
	return openInExplorer(path)
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// platformOpen is implemented per-platform in open_*.go files.
var openInExplorer = func(path string) error {
	return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
}
