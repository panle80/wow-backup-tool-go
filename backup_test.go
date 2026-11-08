package main

import (
	"os"
	"path/filepath"
	"testing"
)

// A restore must never delete the target folders when the archive cannot be
// opened. Previously RestoreBackup removed them before opening the ZIP.
func TestRestoreBackupInvalidZipDoesNotDeleteTarget(t *testing.T) {
	installDir := t.TempDir()
	interfaceDir := filepath.Join(installDir, "Interface")
	if err := os.MkdirAll(interfaceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(interfaceDir, "live.txt")
	if err := os.WriteFile(marker, []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}

	badZip := filepath.Join(t.TempDir(), "bad.zip")
	if err := os.WriteFile(badZip, []byte("not a zip"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := NewBackupManager()
	if _, err := m.RestoreBackup(
		badZip,
		WoWInstallation{Path: installDir},
		[]string{"Interface"},
		false,
		nil,
	); err == nil {
		t.Fatal("RestoreBackup() error = nil, want error")
	}

	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("target file was deleted by a failed restore: %v", err)
	}
}

// A backup that does not contain a selected folder must fail before deleting
// anything, instead of removing the folder and reporting success.
func TestRestoreBackupMissingFolderDoesNotDeleteTarget(t *testing.T) {
	installDir := t.TempDir()
	wtfDir := filepath.Join(installDir, "WTF")
	if err := os.MkdirAll(wtfDir, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(wtfDir, "Config.wtf")
	if err := os.WriteFile(marker, []byte("live"), 0o644); err != nil {
		t.Fatal(err)
	}

	srcDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(srcDir, "Interface"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "Interface", "addon.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := NewBackupManager()
	zipPath, err := m.CreateBackup(
		WoWInstallation{Path: srcDir, Version: "_retail_", DisplayName: "正式服"},
		[]string{"Interface"},
		t.TempDir(),
		nil,
	)
	if err != nil {
		t.Fatalf("CreateBackup() error = %v", err)
	}

	if _, err := m.RestoreBackup(
		zipPath,
		WoWInstallation{Path: installDir},
		[]string{"Interface", "WTF"},
		false,
		nil,
	); err == nil {
		t.Fatal("RestoreBackup() error = nil, want missing-folder error")
	}

	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("WTF was deleted even though it is not in the backup: %v", err)
	}
}
