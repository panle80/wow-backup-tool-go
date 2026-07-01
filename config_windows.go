//go:build windows

package main

import "golang.org/x/sys/windows/registry"

const regKey = `SOFTWARE\pan.wowbackup`
const regValue = "LastOutputDir"

func loadLastOutputDirOS() string {
	k, err := registry.OpenKey(registry.CURRENT_USER, regKey, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer k.Close()
	s, _, err := k.GetStringValue(regValue)
	if err != nil {
		return ""
	}
	return s
}

func saveLastOutputDirOS(dir string) {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, regKey, registry.SET_VALUE)
	if err != nil {
		return
	}
	defer k.Close()
	_ = k.SetStringValue(regValue, dir)
}
