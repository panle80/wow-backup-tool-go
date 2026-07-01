# CLAUDE.md

This file provides guidance to Claude Code when working with this repository.

## Project overview

WoW Backup Tool v2.0 — a desktop app for backing up and restoring World of Warcraft addon data. Cross-platform: Windows (primary) and macOS.

**Tech stack:** Go + Wails v2 (system WebView frontend). Migrated from Python/PyQt6 (v1.x, 66 MB) to Go/Wails (v2.0, ~11 MB single binary, near-instant startup).

## Commands

```bash
# Development (hot-reload frontend + Go)
wails dev

# Windows build → build/bin/WoWBackupTool.exe
wails build -platform windows/amd64 -clean

# macOS build
wails build -platform darwin/arm64 -clean   # Apple Silicon
wails build -platform darwin/amd64 -clean   # Intel

# Type check / lint
go vet ./...
```

## Architecture

```
├── main.go                # Entry point — Wails app, embed frontend, window config
├── app.go                 # App struct — bridge: Go methods → JS callable
│                          #   Async ops use goroutines + runtime.EventsEmit
│                          #   Both backup/restore goroutines have panic recovery
├── detector.go            # WoWDetector — cross-platform detection logic
├── detector_windows.go    # Windows: registry scan + Battle.net.config + product.db
├── detector_other.go      # Non-Windows stubs
├── zipper.go              # ZipManager — ZIP create/extract, manifest, throttled progress
├── backup.go              # BackupManager — orchestrates detect → zip → restore
├── open_windows.go        # Windows: open folder in Explorer
├── open_darwin.go         # macOS: open folder in Finder
├── wails.json             # Wails project config
├── go.mod / go.sum        # Go module
├── appicon.png            # App icon (1024×1024, used by Wails for all platforms)
├── frontend/
│   ├── index.html         # Two-tab layout (创建备份 / 还原备份)
│   ├── package.json       # Vite dev server
│   ├── vite.config.js
│   ├── wailsjs/           # Auto-generated bindings (gitignored, regenerated on build)
│   └── src/
│       ├── main.ts         # TypeScript app logic — two independent panels
│       └── style.css       # Minimal dark theme
└── build/bin/             # Build output
```

## UI layout

Two tabs with consistent structure — WoW install selector at the top of both:

```
Backup tab                     Restore tab
┌────────────────────┐        ┌────────────────────┐
│ 从哪个 WoW 备份     │        │ 还原到哪个 WoW     │  ← WoW selector (same position)
│ [select] [刷新]     │        │ [select] [刷新]     │
├────────────────────┤        ├────────────────────┤
│ 备份哪些内容        │        │ 还原哪些内容        │
│ ☑ Interface        │        │ ☑ Interface        │
│ ☑ WTF              │        │ ☑ WTF              │
│ ☑ Fonts            │        │ ☑ Fonts            │
├────────────────────┤        │ ☑ 保留显示配置      │
│ 保存到              │        ├────────────────────┤
│ [path] [浏览]       │        │ 从哪个备份文件还原   │  ← ZIP file selector
├────────────────────┤        │ [path] [选择]       │
│ [创建备份]          │        ├────────────────────┤
└────────────────────┘        │ [还原备份]          │
                              └────────────────────┘
```

Restore tab: selecting a backup file shows its info (date, size, contained folders) and auto-checks matching folders. Button is disabled until a file is selected.

## Key design decisions

### Frontend ↔ Backend bridge

The `App` struct (`app.go`) exposes exported methods that Wails auto-binds to JS. Generated TypeScript declarations are in `frontend/wailsjs/go/main/App.d.ts`. The frontend (`main.ts`) imports typed bindings — type errors fail at build time.

- **Sync methods** return values directly (e.g. `DetectInstallations()`, `GetBackupInfo()`)
- **Async methods** (`CreateBackup`, `RestoreBackup`) return `void` immediately, spawn goroutines, and emit events: `{backup,restore}:progress`, `:finished`, `:error`
- Both goroutines have `defer/recover` to prevent silent crashes

⚠️ Wails `MessageDialog` with `QuestionDialog` type ignores custom button labels — always returns English `"yes"`/`"no"`. Don't check for Chinese button text.

### Performance

- **Compression level 3** via `compress/flate` registered with `zip.RegisterCompressor` (was 6 in Python). Much faster, negligible size difference for WoW addons (mostly .lua text and pre-compressed assets).
- **Progress throttling** — emits at most every 80ms via `progressThrottler`, instead of per-file IPC calls. Thousands of addon files → ~12 UI updates/sec.

### WoW detection (Windows)

Three-stage fallback:
1. **Registry**: legacy Blizzard keys + Windows Uninstall registry
2. **Battle.net config**: `%APPDATA%\Battle.net\Battle.net.config` JSON; legacy `product.db` via raw byte scan with drive-letter/UNC path validation (no CGO/SQLite dependency)
3. **Fixed paths**: preset list + optional `config.json` overrides

macOS: fixed paths + config.json overrides only.

### Backup format

Standard ZIP with embedded `manifest.json`:
- Fields: `version`, `date`, `toolVersion`, `sourcePlatform`, `platforms`, `files`
- `sourcePlatform` uses `runtime.GOOS` (`"windows"` or `"darwin"`)
- Output: `~/Documents/WoWBackups/WoWBackup_{version}_{timestamp}.zip`

### Mirror restore

Before extracting, target folders are deleted with `os.RemoveAll()`. If `keepLocalConfig=true` and `WTF` is selected, the current `WTF/Config.wtf` is saved to memory and written back after extraction. Controlled by a checkbox in the restore tab (default: checked).

### System file filtering

Skips `.DS_Store`, `Thumbs.db`, `__MACOSX` directories, and `._*` Apple Double files during both backup and extraction. Path-traversal protection via `filepath.Clean` prefix check.

### Window config

- 600×640 default, 520×500 min
- Dark background colour `{11, 13, 15}` matching CSS `--bg` to prevent white flash
- DevTools disabled in production

## Differences from Python v1.x

| Aspect | v1.x (Python) | v2.0 (Go+Wails) |
|---|---|---|
| Runtime | Python + PyQt6 DLLs | Single native binary |
| Size | 66 MB | ~11 MB |
| Startup | 2-4 s | <0.5 s |
| UI | PyQt6 widgets | HTML/CSS/TS via WebView |
| Concurrency | QThread + pyqtSignal | goroutine + Wails events |
| Compression | level 6 | level 3 (registered compressor) |
| Progress | per-file emit | throttled ~80ms |
| Frontend lang | — | TypeScript with auto-generated types |
| Layout | single-page mixed | two tabs, consistent structure |

## Prerequisites

- Go 1.21+
- Node.js 18+ (Vite dev server; not needed at runtime)
- Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- Windows: WebView2 runtime (built into Win10 21H2+)
- macOS: Xcode CLI tools (`xcode-select --install`)
