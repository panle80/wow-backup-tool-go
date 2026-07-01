//go:build darwin

package main

import "os/exec"

func init() {
	openInExplorer = func(path string) error {
		return exec.Command("open", "-R", path).Start()
	}
}
