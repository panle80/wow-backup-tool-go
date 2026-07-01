package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Manifest
// ---------------------------------------------------------------------------

// BackupManifest holds metadata stored inside each backup ZIP.
type BackupManifest struct {
	Version        string            `json:"version"`
	Date           string            `json:"date"`
	ToolVersion    string            `json:"toolVersion"`
	SourcePlatform string            `json:"sourcePlatform"`
	Platforms      []string          `json:"platforms"`
	Files          map[string]string `json:"files"`
}

// NewManifest creates a manifest for the given platform folder list.
func NewManifest(platforms []string) *BackupManifest {
	return &BackupManifest{
		Version:        "1.0",
		ToolVersion:    "2.0.0",
		SourcePlatform: runtime.GOOS,
		Platforms:      platforms,
		Files:          map[string]string{},
	}
}

// ---------------------------------------------------------------------------
// System-file filtering (matches Python version exactly)
// ---------------------------------------------------------------------------

var excludedNames = map[string]bool{
	".DS_Store": true,
	"Thumbs.db": true,
	"__MACOSX":  true,
}

func shouldExclude(name string) bool {
	if excludedNames[name] {
		return true
	}
	if strings.HasPrefix(name, "._") {
		return true
	}
	return false
}

func pathShouldExclude(path string) bool {
	parts := strings.Split(strings.ReplaceAll(path, "\\", "/"), "/")
	for _, p := range parts {
		if p != "" && shouldExclude(p) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// ZipManager
// ---------------------------------------------------------------------------

// ZipManager handles backup creation and extraction.
type ZipManager struct {
	compressLevel int
}

// NewZipManager creates a new ZipManager with fast compression (level 3).
// WoW addons are mostly .lua/.toc text and already-compressed assets,
// so higher levels buy negligible size savings at significant CPU cost.
func NewZipManager() *ZipManager {
	return &ZipManager{compressLevel: 3}
}

// progressThrottler limits callback frequency to avoid flooding the UI.
type progressThrottler struct {
	lastTime time.Time
	minGap   time.Duration
}

func newProgressThrottler(minGap time.Duration) *progressThrottler {
	return &progressThrottler{minGap: minGap}
}

func (t *progressThrottler) emit(fn func(string, int, int), msg string, cur, total int) {
	now := time.Now()
	if now.Sub(t.lastTime) >= t.minGap || cur == 0 || cur >= total {
		t.lastTime = now
		fn(msg, cur, total)
	}
}

// fileEntry describes a single file to be zipped.
type fileEntry struct {
	archivePath string // path inside the ZIP
	diskPath    string // absolute path on disk
	size        int64  // file size in bytes
}

// collectFiles walks the source directories and returns a flat list of files.
func (z *ZipManager) collectFiles(sourcePaths map[string]string) ([]fileEntry, int64, error) {
	var files []fileEntry
	var totalSize int64

	for archiveName, actualPath := range sourcePaths {
		err := filepath.Walk(actualPath, func(fpath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip excluded directories (don't walk into them)
			if info.IsDir() && shouldExclude(info.Name()) {
				return filepath.SkipDir
			}
			// Skip excluded files
			if !info.IsDir() && shouldExclude(info.Name()) {
				return nil
			}
			if info.IsDir() {
				return nil
			}

			rel, err := filepath.Rel(actualPath, fpath)
			if err != nil {
				return err
			}

			archiveRel := filepath.Join(archiveName, rel)
			// ZIP paths always use forward slashes
			archiveRel = strings.ReplaceAll(archiveRel, "\\", "/")
			files = append(files, fileEntry{
				archivePath: archiveRel,
				diskPath:    fpath,
				size:        info.Size(),
			})
			totalSize += info.Size()
			return nil
		})
		if err != nil {
			return nil, 0, fmt.Errorf("walk %s: %w", actualPath, err)
		}
	}
	return files, totalSize, nil
}

// CreateBackup writes a ZIP file at outputPath containing the given folders.
// progressFn is called with (message, currentBytes, totalBytes).
func (z *ZipManager) CreateBackup(
	sourcePaths map[string]string,
	outputPath string,
	manifest *BackupManifest,
	progressFn func(string, int, int),
) error {
	manifest.Date = time.Now().UTC().Format(time.RFC3339)

	files, totalSize, err := z.collectFiles(sourcePaths)
	if err != nil {
		return err
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create zip file: %w", err)
	}
	defer outFile.Close()

	zipWriter := zip.NewWriter(outFile)
	defer zipWriter.Close()

	throttle := newProgressThrottler(80 * time.Millisecond)
	var currentSize int64
	for _, fe := range files {
		if progressFn != nil {
			throttle.emit(progressFn, fe.archivePath, int(currentSize), int(totalSize))
		}

		writer, err := zipWriter.Create(fe.archivePath)
		if err != nil {
			return fmt.Errorf("zip create %s: %w", fe.archivePath, err)
		}

		src, err := os.Open(fe.diskPath)
		if err != nil {
			return fmt.Errorf("open %s: %w", fe.diskPath, err)
		}

		_, err = io.Copy(writer, src)
		src.Close()
		if err != nil {
			return fmt.Errorf("write %s: %w", fe.archivePath, err)
		}

		currentSize += fe.size
	}

	// Embed manifest.json inside the ZIP
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	w, err := zipWriter.Create("manifest.json")
	if err != nil {
		return fmt.Errorf("create manifest entry: %w", err)
	}
	if _, err := w.Write(manifestData); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	return nil
}

// ExtractBackup extracts a ZIP to targetPath, optionally filtering by top-level folders.
func (z *ZipManager) ExtractBackup(
	backupPath, targetPath string,
	selectedFolders []string,
	progressFn func(string, int, int),
) (*BackupManifest, error) {
	reader, err := zip.OpenReader(backupPath)
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}
	defer reader.Close()

	// Resolve target to absolute and clean
	targetPath, err = filepath.Abs(targetPath)
	if err != nil {
		return nil, fmt.Errorf("resolve target path: %w", err)
	}

	// Filter files
	allFiles := make([]*zip.File, 0, len(reader.File))
	for _, f := range reader.File {
		if f.Name == "manifest.json" {
			allFiles = append(allFiles, f)
			continue
		}
		if pathShouldExclude(f.Name) {
			continue
		}
		if len(selectedFolders) > 0 {
			include := false
			for _, folder := range selectedFolders {
				prefix := folder + "/"
				if strings.HasPrefix(f.Name, prefix) {
					include = true
					break
				}
			}
			if !include {
				continue
			}
		}
		allFiles = append(allFiles, f)
	}

	total := len(allFiles)
	var manifest *BackupManifest
	throttle := newProgressThrottler(80 * time.Millisecond)

	for i, f := range allFiles {
		if progressFn != nil {
			throttle.emit(progressFn, f.Name, i, total)
		}

		// Path-traversal protection
		destPath := filepath.Join(targetPath, f.Name)
		destPath = filepath.Clean(destPath)
		if !strings.HasPrefix(destPath, targetPath+string(os.PathSeparator)) && destPath != targetPath {
			return nil, fmt.Errorf("安全警告: 路径穿越攻击 — %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(destPath, 0o755)
			continue
		}

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return nil, fmt.Errorf("mkdir %s: %w", filepath.Dir(destPath), err)
		}

		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("open zip entry %s: %w", f.Name, err)
		}

		outFile, err := os.Create(destPath)
		if err != nil {
			rc.Close()
			return nil, fmt.Errorf("create %s: %w", destPath, err)
		}

		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if err != nil {
			return nil, fmt.Errorf("write %s: %w", destPath, err)
		}
	}

	// Read manifest
	for _, f := range reader.File {
		if f.Name == "manifest.json" {
			rc, err := f.Open()
			if err == nil {
				defer rc.Close()
				var fm BackupManifest
				if json.NewDecoder(rc).Decode(&fm) == nil {
					manifest = &fm
				}
			}
			break
		}
	}

	if manifest == nil {
		manifest = &BackupManifest{}
	}
	return manifest, nil
}

// GetBackupInfo reads metadata from a backup ZIP without extracting it.
func (z *ZipManager) GetBackupInfo(backupPath string) (map[string]interface{}, error) {
	info := map[string]interface{}{
		"filePath": backupPath,
		"fileName": filepath.Base(backupPath),
	}

	stat, err := os.Stat(backupPath)
	if err != nil {
		return nil, fmt.Errorf("stat backup: %w", err)
	}

	reader, err := zip.OpenReader(backupPath)
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}
	defer reader.Close()

	fileCount := 0
	var totalSize int64
	folders := make(map[string]bool)

	for _, f := range reader.File {
		if f.Name == "manifest.json" {
			continue
		}
		if pathShouldExclude(f.Name) {
			continue
		}
		fileCount++
		totalSize += int64(f.UncompressedSize64)

		// Extract top-level folder
		parts := strings.SplitN(f.Name, "/", 2)
		if len(parts) > 0 && parts[0] != "" && !shouldExclude(parts[0]) {
			folders[parts[0]] = true
		}
	}

	topFolders := make([]string, 0, len(folders))
	for f := range folders {
		topFolders = append(topFolders, f)
	}

	info["fileCount"] = fileCount
	info["totalContentSize"] = totalSize
	info["fileSize"] = stat.Size()
	info["formattedSize"] = formatSize(stat.Size())
	info["topFolders"] = topFolders

	// Read manifest
	for _, f := range reader.File {
		if f.Name == "manifest.json" {
			rc, err := f.Open()
			if err == nil {
				defer rc.Close()
				var fm BackupManifest
				if json.NewDecoder(rc).Decode(&fm) == nil {
					info["manifest"] = fm
					info["version"] = fm.Version
					info["date"] = fm.Date
				}
			}
			break
		}
	}

	return info, nil
}

// ListBackups returns metadata for all WoW backup ZIPs in a directory.
func (z *ZipManager) ListBackups(dir string) ([]map[string]interface{}, error) {
	if !dirExists(dir) {
		return nil, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}

	var backups []map[string]interface{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, "WoWBackup_") || !strings.HasSuffix(name, ".zip") {
			continue
		}

		fp := filepath.Join(dir, name)
		info, err := z.GetBackupInfo(fp)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[zipper] failed to read %s: %v\n", fp, err)
			continue
		}
		backups = append(backups, info)
	}

	// Sort by date descending
	for i := 0; i < len(backups); i++ {
		for j := i + 1; j < len(backups); j++ {
			di, _ := backups[i]["date"].(string)
			dj, _ := backups[j]["date"].(string)
			if dj > di {
				backups[i], backups[j] = backups[j], backups[i]
			}
		}
	}

	return backups, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func formatSize(size int64) string {
	const unit = int64(1024)
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := unit, 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	suffixes := []string{"KB", "MB", "GB", "TB"}
	if exp >= len(suffixes) {
		exp = len(suffixes) - 1
		div = 1
		for i := 0; i <= exp; i++ {
			div *= unit
		}
	}
	return fmt.Sprintf("%.1f %s", float64(size)/float64(div), suffixes[exp])
}
