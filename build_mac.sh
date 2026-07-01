#!/bin/bash
# ============================================================
#  WoW Backup Tool — macOS Build Script
#  Requires: Go 1.21+, Node.js 18+, Wails CLI, Xcode CLI tools
#  First-time setup:
#    go install github.com/wailsapp/wails/v2/cmd/wails@latest
#    wails doctor
# ============================================================

set -e

echo "[1/3] Installing frontend dependencies..."
cd frontend && npm install && cd ..

echo "[2/3] Building macOS .app bundle (Apple Silicon)..."
wails build -platform darwin/arm64 -clean

echo "[3/3] Building macOS .app bundle (Intel)..."
wails build -platform darwin/amd64 -clean

echo ""
echo "Build complete!"
echo "  Apple Silicon: build/bin/WoWBackupTool.app"
echo "  Intel:         build/bin/WoWBackupTool (darwin/amd64)"
echo ""
echo "To create a DMG:  hdiutil create -volname 'WoW Backup Tool' -srcfolder"
echo "  'build/bin/WoWBackupTool.app' -ov -format UDZO 'WoWBackupTool.dmg'"
