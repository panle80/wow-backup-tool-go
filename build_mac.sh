#!/bin/bash
# ============================================================
#  WoW Backup Tool — macOS Build Script
#  Requires: Go 1.21+, Node.js 18+, Wails CLI, Xcode CLI tools
#  First-time setup:
#    go install github.com/wailsapp/wails/v2/cmd/wails@latest
#    wails doctor
#
#  Note: Apple Silicon only (M1/M2/M3). Intel Mac 不支持。
# ============================================================

set -e

echo "[1/2] Installing frontend dependencies..."
cd frontend && npm install && cd ..

echo "[2/2] Building macOS .app bundle (Apple Silicon)..."
wails build -platform darwin/arm64 -clean

echo ""
echo "Build complete!"
echo "  Apple Silicon: build/bin/WoWBackupTool.app"
echo ""
echo "To create a DMG:  hdiutil create -volname 'WoW Backup Tool' -srcfolder"
echo "  'build/bin/WoWBackupTool.app' -ov -format UDZO 'WoWBackupTool.dmg'"
