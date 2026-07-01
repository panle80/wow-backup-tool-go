//go:build darwin

package main

import "os/exec"

func loadLastOutputDirOS() string {
	out, err := exec.Command("defaults", "read", "pan.wowbackup", "LastOutputDir").Output()
	if err != nil {
		return ""
	}
	// Trim trailing newline
	s := string(out)
	if len(s) > 0 && s[len(s)-1] == '\n' {
		s = s[:len(s)-1]
	}
	return s
}

func saveLastOutputDirOS(dir string) {
	_ = exec.Command("defaults", "write", "pan.wowbackup", "LastOutputDir", dir).Run()
}
