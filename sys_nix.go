// +build !windows

package main

func GetDrives() (map[rune]bool, error) {
	return make(map[rune]bool), nil
}
