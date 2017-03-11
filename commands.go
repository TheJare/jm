package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func RunCommand(command string, args ...string) error {
	var ret []string
	shellarg := "-c"
	shell, ok := os.LookupEnv("COMSPEC")
	if ok {
		shellarg = "/C"
	} else {
		shell, ok = os.LookupEnv("SHELL")
		if !ok {
			shell = "/bin/sh"
		}
	}
	finalargs := append([]string{shell, shellarg, command}, args...)
	cmd := exec.Command(shell, finalargs...)

	outp, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("Failed to run command: %v", err)
	}
	cmd.Start()
	scanner := bufio.NewScanner(outp)
	for scanner.Scan() {
		line := scanner.Text()
		ret = append(ret, line)
	}
	fmt.Fprintf(os.Stderr, strings.Join(ret, "\n"))
	return cmd.Wait()
}
