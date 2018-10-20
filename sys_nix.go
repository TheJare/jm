// +build !windows

package main

import "os/exec"

// GetDrives returns a map of drive letters. *nix systems dont have drives, so empty list
func GetDrives() (map[rune]bool, error) {
	return make(map[rune]bool), nil
}

// SetProcCmdline dummy, only needed on Windows
func SetProcCmdline(cmd *exec.Cmd, cmdline string) {
}
