@echo off
REM WoW Backup Tool - Windows Build Script

echo [1/3] Copying icon...
copy /Y huifubeifen.png build\appicon.png >nul

echo [2/3] Clearing cached icon...
if exist build\windows\ rd /s /q build\windows

echo [3/3] Building...
wails build -platform windows/amd64

echo.
echo Done: build\bin\WoWBackupTool.exe
echo If the exe icon looks wrong, clear Windows icon cache:
echo   del /f %LOCALAPPDATA%\IconCache.db ^& taskkill /f /im explorer.exe ^& start explorer.exe
