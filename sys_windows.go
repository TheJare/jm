// +build windows

package main

import (
	"syscall"
)

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
