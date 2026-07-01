//go:build !windows

package main

// Stubs for platforms that don't have Windows registry / Battle.net.
func (d *WoWDetector) detectWindowsRegistry() []WoWInstallation { return nil }
func (d *WoWDetector) detectBattleNet() []WoWInstallation       { return nil }
