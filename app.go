package main

import (
	"context"
	"fmt"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ---------------------------------------------------------------------------
// Progress data sent to the frontend via events
// ---------------------------------------------------------------------------

type ProgressData struct {
	Message string `json:"message"`
	Current int    `json:"current"`
	Total   int    `json:"total"`
}

// ---------------------------------------------------------------------------
// App — Wails application struct (all exported methods are callable from JS)
// ---------------------------------------------------------------------------

type App struct {
	ctx     context.Context
	manager *BackupManager
}

// NewApp creates the application instance.
func NewApp() *App {
	return &App{
		manager: NewBackupManager(),
	}
}

// startup is called by Wails when the app starts. We store the context for
// runtime calls (events, dialogs, etc.).
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// shutdown is called when the app is about to quit.
func (a *App) shutdown(ctx context.Context) {
	// Cleanup if needed
}

// ---------------------------------------------------------------------------
// Detection (synchronous — return immediately)
// ---------------------------------------------------------------------------

// DetectInstallations returns all detected WoW installations.
func (a *App) DetectInstallations() []WoWInstallation {
	return a.manager.DetectInstallations()
}

// DetectFromPath tries to detect a WoW installation at the given path.
// Returns nil if nothing was found.
func (a *App) DetectFromPath(path string) *WoWInstallation {
	return a.manager.DetectFromPath(path)
}

// ValidateInstallation checks which key folders exist under the given path.
func (a *App) ValidateInstallation(path string) map[string]bool {
	return a.manager.ValidateInstallation(path)
}

// ---------------------------------------------------------------------------
// Backup (async — fires progress events, result delivered via event)
// ---------------------------------------------------------------------------

// CreateBackup starts a backup operation. Progress is reported via the
// "backup:progress" event; completion via "backup:finished"; errors via
// "backup:error".
func (a *App) CreateBackup(installPath, version, displayName string, selectedFolders []string, outputDir string) {
	inst := WoWInstallation{
		Path:        installPath,
		Version:     version,
		DisplayName: displayName,
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				runtime.EventsEmit(a.ctx, "backup:error", fmt.Sprintf("panic: %v", r))
			}
		}()

		path, err := a.manager.CreateBackup(inst, selectedFolders, outputDir,
			func(msg string, cur, total int) {
				runtime.EventsEmit(a.ctx, "backup:progress", ProgressData{
					Message: msg, Current: cur, Total: total,
				})
			},
		)
		if err != nil {
			runtime.EventsEmit(a.ctx, "backup:error", err.Error())
		} else {
			runtime.EventsEmit(a.ctx, "backup:finished", path)
		}
	}()
}

// ---------------------------------------------------------------------------
// Restore (async — fires progress events, result delivered via event)
// ---------------------------------------------------------------------------

// RestoreBackup starts a restore operation. Progress via "restore:progress",
// completion via "restore:finished", errors via "restore:error".
func (a *App) RestoreBackup(backupPath, installPath, version, displayName, keepConfig string, selectedFolders []string) {
	keepLocalConfig := keepConfig == "true"

	inst := WoWInstallation{
		Path:        installPath,
		Version:     version,
		DisplayName: displayName,
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				runtime.EventsEmit(a.ctx, "restore:error", fmt.Sprintf("panic: %v", r))
			}
		}()

		result, err := a.manager.RestoreBackup(backupPath, inst, selectedFolders, keepLocalConfig,
			func(msg string, cur, total int) {
				runtime.EventsEmit(a.ctx, "restore:progress", ProgressData{
					Message: msg, Current: cur, Total: total,
				})
			},
		)
		if err != nil {
			runtime.EventsEmit(a.ctx, "restore:error", err.Error())
		} else {
			runtime.EventsEmit(a.ctx, "restore:finished", result)
		}
	}()
}

// ---------------------------------------------------------------------------
// Backup info / listing (synchronous)
// ---------------------------------------------------------------------------

// GetBackupInfo reads metadata from a backup ZIP file.
func (a *App) GetBackupInfo(backupPath string) (map[string]interface{}, error) {
	return a.manager.GetBackupInfo(backupPath)
}

// ListBackups returns metadata for all backup ZIPs in the given directory.
func (a *App) ListBackups(directory string) ([]map[string]interface{}, error) {
	return a.manager.ListBackups(directory)
}

// ---------------------------------------------------------------------------
// Dialogs
// ---------------------------------------------------------------------------

// SelectDirectory opens a native directory picker and returns the selected path.
func (a *App) SelectDirectory(title string) (string, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
	})
	if err != nil {
		return "", err
	}
	return dir, nil
}

// SelectFile opens a native file picker for .zip files.
func (a *App) SelectFile(title string) (string, error) {
	file, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
		Filters: []runtime.FileFilter{
			{DisplayName: "ZIP 文件 (*.zip)", Pattern: "*.zip"},
			{DisplayName: "所有文件 (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil {
		return "", err
	}
	return file, nil
}

// OpenFolder opens the given path in the system file manager.
func (a *App) OpenFolder(path string) error {
	return a.manager.OpenFolder(path)
}

// DefaultOutputPath returns the default backup directory.
func (a *App) DefaultOutputPath() string {
	return DefaultOutputDir()
}

// ShowMessage shows a native message dialog.
func (a *App) ShowMessage(dialogType, title, message string) (string, error) {
	switch dialogType {
	case "info":
		_, err := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:    runtime.InfoDialog,
			Title:   title,
			Message: message,
		})
		return "ok", err
	case "warning":
		_, err := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:    runtime.WarningDialog,
			Title:   title,
			Message: message,
		})
		return "ok", err
	case "error":
		_, err := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:    runtime.ErrorDialog,
			Title:   title,
			Message: message,
		})
		return "ok", err
	case "question":
		resp, err := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:    runtime.QuestionDialog,
			Title:   title,
			Message: message,
			Buttons: []string{"是", "否"},
		})
		return resp, err
	default:
		return "", fmt.Errorf("unknown dialog type: %s", dialogType)
	}
}
