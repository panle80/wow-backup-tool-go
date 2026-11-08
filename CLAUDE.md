# CLAUDE.md

This file provides guidance to Claude Code when working with this repository.

## Project overview

WoW Backup Tool v2.1.0 — a desktop app for backing up and restoring World of Warcraft addon data. Cross-platform: Windows (primary) and macOS.

**Tech stack:** Go + Wails v2 (system WebView frontend). Migrated from Python/PyQt6 (v1.x, 66 MB) to Go/Wails (v2.0, ~11 MB single binary, near-instant startup).

## Commands

```bash
# Development (hot-reload frontend + Go)
wails dev

# Windows build → build/bin/WoWBackupTool.exe
wails build -platform windows/amd64 -clean

# macOS build (Apple Silicon only)
wails build -platform darwin/arm64 -clean   # Apple Silicon

# Type check / lint
go vet ./...

# Frontend type check + production build (tsc --noEmit && vite build)
cd frontend && npm run build

# Go tests (main.go embeds frontend/dist, so build the frontend first on a
# clean checkout)
go test ./...
```

## Architecture

```
├── main.go                # Entry point — Wails app, embed frontend, window config, AppVersion
├── app.go                 # App struct — bridge: Go methods → JS callable
│                          #   Async ops use goroutines + runtime.EventsEmit
│                          #   Both backup/restore goroutines have panic recovery
├── detector.go            # WoWDetector — detection logic + config.json custom paths
├── detector_test.go       # Unit tests for version-folder detection
├── detector_windows.go    # Windows: registry scan + Battle.net.config + product.db
├── detector_other.go      # Non-Windows stubs
├── zipper.go              # ZipManager — ZIP create/extract, manifest, throttled progress
├── backup.go              # BackupManager — orchestrates detect → zip → restore
│                          #   Restore validates the ZIP/folders before deleting anything
├── backup_test.go         # Restore safety tests (invalid ZIP / missing folder)
├── version_test.go        # Keeps AppVersion, wails.json and package.json in sync
├── platform_windows.go    # Windows: registry output-dir persistence + open in Explorer
├── platform_darwin.go     # macOS: UserDefaults output-dir persistence + open in Finder
├── platform_other.go      # Other platforms: no-op output-dir stubs
├── wails.json             # Wails project config
├── go.mod / go.sum        # Go module
├── build.bat               # Windows build script (copies icon + wails build)
├── huifubeifen.png          # App icon source (256×256, copied to build/appicon.png for Wails)
├── frontend/
│   ├── index.html         # Global game selector + two-tab layout (创建备份 / 还原备份)
│   ├── package.json       # Vite dev server; build runs tsc --noEmit first
│   ├── tsconfig.json
│   ├── vite.config.js
│   ├── wailsjs/           # Auto-generated bindings (gitignored, regenerated on build)
│   └── src/
│       ├── main.ts         # TypeScript app logic — shared game selector + two panels
│       └── style.css       # Minimal dark theme
└── build/bin/             # Build output
```

## UI layout

One global game selector above the tabs — backup and restore always share the
same `currentInstall`, so there is no per-tab WoW dropdown anymore:

```
┌──────────────────────────────┐
│ WoW Backup Tool    © 2026 …  │  header
├──────────────────────────────┤
│ ① 当前游戏          [切换]    │  game card: name, path, ✔ Interface ✔ WTF
├──────────────────────────────┤
│ [创建备份] [还原备份]          │  tabs
├──────────────────────────────┤
│ ② 备份/还原哪些内容            │
│ ☑ Interface ☑ WTF ☑ Fonts    │
│ ☑ 保留显示配置 (restore only)  │
├──────────────────────────────┤
│ ③ 保存到 / 从哪个备份文件还原   │
│ [path] [浏览/选择文件]         │
├──────────────────────────────┤
│ [创建备份 / 还原备份]          │
└──────────────────────────────┘
```

Clicking “切换” opens a modal listing every detected installation with a
可用/不完整 badge. “重新检测” re-runs `DetectInstallations()`; “浏览其他目录...”
calls `DetectAllFromPath(path)` so a root folder containing several version
folders (`_retail_` + `_classic_` + `_forever_`) contributes all of them to the
picker instead of just the first.

Restore tab: selecting a backup file shows its info (date, size, contained folders) and auto-checks matching folders. Button is disabled until a file is selected.

## Key design decisions

### Frontend ↔ Backend bridge

The `App` struct (`app.go`) exposes exported methods that Wails auto-binds to JS. Generated TypeScript declarations are in `frontend/wailsjs/go/main/App.d.ts`. The frontend (`main.ts`) imports typed bindings, and `npm run build` runs `tsc --noEmit` before Vite — type errors fail at build time.

- **Sync methods** return values directly (e.g. `DetectInstallations()`, `DetectAllFromPath()`, `GetBackupInfo()`)
- **Async methods** (`CreateBackup`, `RestoreBackup`) return `void` immediately, spawn goroutines, and emit events: `{backup,restore}:progress`, `:finished`, `:error`
- Both goroutines have `defer/recover` to prevent silent crashes

⚠️ Wails `MessageDialog` with `QuestionDialog` type ignores custom button labels — always returns English `"yes"`/`"no"`. Don't check for Chinese button text.

### Versioning

`AppVersion` in `main.go` is the single Go-side source of truth — the window
title and the backup manifest's `toolVersion` both use it. `version_test.go`
asserts that `wails.json` (`info.productVersion`) and `frontend/package.json`
match, so bumping the version means updating those three places. CI runs
`go vet` + `go test` on every push before building.

### Performance

- **Default compression** (level 6) — Go's `archive/zip` uses Deflate internally. Custom compression level was attempted via `RegisterCompressor` but Go 1.26 panics on re-registration of the built-in Deflate method.
- **Progress throttling** — emits at most every 80ms via `progressThrottler`, instead of per-file IPC calls. Thousands of addon files → ~12 UI updates/sec.

### WoW detection (Windows)

Version folders are recognised by the `_..._` naming pattern. `versionNames` in
`detector.go` supplies display names: `_retail_` (正式服), `_classic_` (经典怀旧服),
`_classic_era_` (经典旧世), `_classic_titan_` (大灾变怀旧服), `_forever_` (无限服),
`_classic_ptr_`, `_ptr_`, `_beta_`. Unknown versions are detected as well and shown
with the raw folder name; add them to `versionNames` to give them a Chinese name.

When scanning a directory (`detectFromPaths`, `DetectAllFromPath` Case 2), unknown
`_..._` folders without Interface/WTF/Fonts are skipped so stray `_old_`/`_backup_`
folders cannot be offered (and auto-selected) as a restore target. Selecting a
version folder explicitly is always accepted — a freshly installed game has no
data folders yet but must still be restorable.

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

### App icon

Wails v2.12 reads the source icon from `build/appicon.png` (NOT the project root `appicon.png`).
The first build copies Wails' embedded default (blue "W") into `build/appicon.png` as a starting
point, and all subsequent builds reuse that cached copy.

To use a custom icon:

```bash
# 1. Copy your PNG into the build directory (256×256 or 1024×1024 RGBA recommended)
cp your_icon.png build/appicon.png

# 2. Delete the cached windows directory so Wails regenerates icon.ico from your PNG
rm -rf build/windows

# 3. Normal build
wails build -platform windows/amd64
```

Wails skips `icon.ico` regeneration if `build/windows/icon.ico` already exists — so deleting
the entire `build/windows/` directory before building is the safest approach.

**Windows icon cache:** After a fresh build, Windows Explorer may still show the old W icon
due to the system icon cache (Issue #1431). If the Properties dialog also shows the wrong icon,
clear the cache and restart Explorer:

```bash
rm -f "$env:LOCALAPPDATA\\IconCache.db"
powershell -Command "Stop-Process -Name explorer -Force"
```

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
