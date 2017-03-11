package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func RunCommand(command string, args ...string) error {
	var ret []string
	var finalargs []string
	shell, ok := os.LookupEnv("COMSPEC")
	if ok {
		finalargs = append([]string{"/C"}, command)
		finalargs = append(finalargs, args...)
	} else {
		shell, ok = os.LookupEnv("SHELL")
		if !ok {
			shell = "/bin/sh"
		}
		for i, v := range args {
			args[i] = strconv.Quote(v)
		}
		args = append([]string{command}, args...)
		finalargs = []string{"-c", strings.Join(args, " ") + " 2>&1"}
	}
	//fmt.Fprintf(os.Stderr, "Running command:\n>%s<\n>>%s<<\n", shell, strings.Join(finalargs, "<<\n>>"))
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
	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("%s: %s", err, strings.Join(ret, "\n"))
	}
	return nil
}

func CommandCopy(src string, dst string) error {
	dst = filepath.Clean(dst)
	if dst[len(dst)-1] == os.PathSeparator {
		return fmt.Errorf("Copy to root folder %s not allowed for safety", dst)
	}
	dst += string(os.PathSeparator)
	// Many safety checks to perform here...
	if runtime.GOOS == "windows" {
		return RunCommand("copy", "/B", "/Y", "/L", src, dst)
	}
	return RunCommand("cp", "-R", src, dst)
}

func CommandMove(src string, dst string) error {
	dst = filepath.Clean(dst)
	if dst[len(dst)-1] == os.PathSeparator {
		return fmt.Errorf("Move to root folder %s not allowed for safety", dst)
	}
	dst += string(os.PathSeparator)
	// Many safety checks to perform here...
	if runtime.GOOS == "windows" {
		return RunCommand("move", "/Y", src, dst)
	}
	return RunCommand("mv", "-f", src, dst)
}

func CommandDelete(src string) error {
	// Many safety checks to perform here...
	if runtime.GOOS == "windows" {
		return RunCommand("del", "/S", src)
	}
	return RunCommand("rm", "-rf", src)
}
