# CLAUDE.md

This file provides guidance to Claude Code when working with this repository.

## Project overview

WoW Backup Tool v2.0 — a desktop app for backing up and restoring World of Warcraft addon data. Cross-platform: Windows (primary) and macOS.

**Tech stack:** Go + Wails v2 (system WebView frontend). Migrated from Python/PyQt6 (v1.x, 66 MB) to Go/Wails (v2.0, ~11 MB single binary, near-instant startup).

## Commands

```bash
# Development (hot-reload frontend + Go, shows console errors)
wails dev

# Windows build → build/bin/WoWBackupTool.exe
wails build -platform windows/amd64 -clean

# macOS build
wails build -platform darwin/arm64 -clean   # Apple Silicon
wails build -platform darwin/amd64 -clean   # Intel

# Type check / lint
go vet ./...

# Install Wails CLI (one-time)
go install github.com/wailsapp/wails/v2/cmd/wails@latest
wails doctor
```

## Architecture

```
├── main.go                # Entry point — Wails app, embed frontend, window config
├── app.go                 # App struct — bridge: Go methods → JS callable
│                          #   Async ops use goroutines + runtime.EventsEmit
├── detector.go            # WoWDetector — cross-platform detection logic
├── detector_windows.go    # Windows: registry scan + Battle.net.config
├── detector_other.go      # Non-Windows stubs
├── zipper.go              # ZipManager — ZIP create/extract, manifest, throttled progress
├── backup.go              # BackupManager — orchestrates detect → zip → restore
├── open_windows.go        # Windows: open folder in Explorer
├── open_darwin.go         # macOS: open folder in Finder
├── wails.json             # Wails project config
├── go.mod / go.sum        # Go module
├── appicon.png            # App icon (1024×1024, used by Wails for all platforms)
├── frontend/
│   ├── index.html         # Two-tab layout (备份 / 还原)
│   ├── package.json       # Vite dev server
│   ├── vite.config.js
│   ├── wailsjs/           # Auto-generated bindings (gitignored)
│   └── src/
│       ├── main.ts         # TypeScript app logic — two independent panels
│       └── style.css       # Minimal dark theme
└── build/bin/             # Build output
```

## Key design decisions

### UI: Tabbed layout

Two independent tabs with consistent top-down flow:

- **备份 tab**: 来源(WoW) → 内容(文件夹) → 去向(保存路径) → 按钮
- **还原 tab**: 来源(ZIP) → 内容(文件夹) → 去向(WoW) → 按钮

Restore tab shows backup file info (date, size, contained folders) on selection, auto-checks matching folder checkboxes. Progress bar shows throttled file names (every ~80ms).

### Frontend ↔ Backend bridge

The `App` struct (`app.go`) exposes exported methods that Wails auto-binds to JS. Generated TypeScript declarations are in `frontend/wailsjs/go/main/App.d.ts`. The frontend (`main.ts`) imports typed bindings — type errors fail at build time.

- **Sync methods** return values directly (e.g. `DetectInstallations()`, `GetBackupInfo()`)
- **Async methods** (`CreateBackup`, `RestoreBackup`) return `void` immediately, then emit events: `{backup,restore}:progress`, `:finished`, `:error`

⚠️ Wails `MessageDialog` with `QuestionDialog` type ignores custom button labels on Chinese Windows — always returns English `"yes"`/`"no"`. Don't check for Chinese button text.

### Performance

- **Compression level 3** (was 6 in Python) — much faster, negligible size difference for WoW addons (mostly .lua text and pre-compressed assets)
- **Progress throttling** — emits at most every 80ms via `progressThrottler`, instead of per-file IPC calls. Thousands of addon files → ~12 UI updates/sec

### WoW detection (Windows)

Three-stage fallback:
1. **Registry**: legacy Blizzard keys + Windows Uninstall registry (`HKLM\...\Uninstall\World of Warcraft *` → `InstallLocation`)
2. **Battle.net config**: `%APPDATA%\Battle.net\Battle.net.config` JSON; legacy `product.db` via raw byte scan (no CGO/SQLite dependency)
3. **Fixed paths**: preset list + optional `config.json` overrides

macOS: fixed paths + config.json overrides only.

### Backup format

Standard ZIP with embedded `manifest.json`:
- Fields: `version`, `date`, `toolVersion`, `sourcePlatform`, `platforms`, `files`
- Output: `~/Documents/WoWBackups/WoWBackup_{version}_{timestamp}.zip`

### Mirror restore

Before extracting, target folders are deleted. If `keepLocalConfig=true` and `WTF` is selected, the current `WTF/Config.wtf` is saved to memory and written back after extraction.

### System file filtering

Skips `.DS_Store`, `Thumbs.db`, `__MACOSX` directories, and `._*` Apple Double files. Path-traversal protection via `filepath.Clean` prefix check.

## Differences from Python v1.x

| Aspect | v1.x (Python) | v2.0 (Go+Wails) |
|---|---|---|
| Runtime | Python + PyQt6 DLLs | Single native binary |
| Size | 66 MB | ~11 MB |
| Startup | 2-4 s | <0.5 s |
| UI | PyQt6 widgets | HTML/CSS/TS via WebView |
| Concurrency | QThread + pyqtSignal | goroutine + Wails events |
| Compression | level 6 | level 3 (throttled) |
| Frontend lang | — | TypeScript with auto-generated types |

## Prerequisites

- Go 1.21+
- Node.js 18+ (Vite dev server; not needed at runtime)
- Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- Windows: WebView2 runtime (built into Win10 21H2+)
- macOS: Xcode CLI tools (`xcode-select --install`)
