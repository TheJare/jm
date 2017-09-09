// +build windows

package main

import (
	"os/exec"
	"syscall"
)

// GetDrives returns a map of drive letters (uppercase) to boolean indicating if it's present or not
func GetDrives() (map[rune]bool, error) {
	drives := make(map[rune]bool)
	kernel32, err := syscall.LoadDLL("kernel32.dll")
	if err != nil {
		return drives, err
	}
	getLogicalDrives, err := kernel32.FindProc("GetLogicalDrives")
	if err != nil {
		return drives, err
	}
	bitmask, _, _ := getLogicalDrives.Call()

	for i := 'A'; i <= 'Z'; i++ {
		if (bitmask & 1) != 0 {
			drives[i] = true
		}
		bitmask >>= 1
	}
	return drives, nil
}

// SetProcCmdline sets the command line on windows to the string,
// which will make the process ignore the array of arguments given
// We need this on windows to avoid MSVCRT quoting since we are
// actually calling CMD, not a random MSVCRT program.
// See http://daviddeley.com/autohotkey/parameters/parameters.htm#WIN
func SetProcCmdline(cmd *exec.Cmd, cmdline string) {
	cmd.SysProcAttr = new(syscall.SysProcAttr)
	cmd.SysProcAttr.CmdLine = cmdline
}
