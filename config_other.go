//go:build !windows && !darwin

package main

func loadLastOutputDirOS() string  { return "" }
func saveLastOutputDirOS(dir string) {}
