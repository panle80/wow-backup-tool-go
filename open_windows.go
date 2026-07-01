//go:build windows

package main

import (
	"os/exec"
)

func init() {
	openInExplorer = func(path string) error {
		return exec.Command("explorer", "/select,"+path).Start()
	}
}
