@echo off
REM ============================================================
REM  WoW Backup Tool — Windows Build Script
REM  需要: Go 1.21+, Node.js 18+, Wails CLI
REM  首次使用请运行:
REM    go install github.com/wailsapp/wails/v2/cmd/wails@latest
REM    wails doctor
REM ============================================================

echo [1/2] Installing frontend dependencies...
cd frontend
call npm install
cd ..

echo [2/2] Building Windows binary...
wails build -platform windows/amd64 -nsis -clean

echo.
echo Build complete! Output is in build\bin\
pause
