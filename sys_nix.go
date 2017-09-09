// +build !windows

package main

import "os/exec"

func GetDrives() (map[rune]bool, error) {
	return make(map[rune]bool), nil
}

// SetProcCmdline dummy, only needed on Windows
func SetProcCmdline(cmd *exec.Cmd, cmdline string) {
}
