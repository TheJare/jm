// Copyright 2017 Javier Arevalo <jare@iguanademos.com>

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

// http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Cross platform command execution and some file system operations

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

// RunShell runs an interactive shell.
// Since the current program is still running, so may be
// its coroutines, which will trigger all sorts of weirdness
// For example, Ctrl-C may be caught by our Go runtime, killing us
// but leaving the spawned shell still running, with I/O shared with
// our parent (possibly another shell!). That's pretty ugly.
func RunShell(cwd string) error {
	attr := os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
		Dir:   cwd,
	}
	var args []string
	shell, ok := os.LookupEnv("COMSPEC")
	if !ok {
		args = []string{"-i"}
		shell, ok = os.LookupEnv("SHELL")
		if !ok {
			shell = "/bin/sh"
		}
	}
	process, err := os.StartProcess(shell, args, &attr)
	if err != nil {
		return err
	}
	state, err := process.Wait()
	if err != nil {
		return err
	}
	if state.Success() {
		return nil
	}
	return fmt.Errorf("<< Exited shell: %s", state.String())
}

// RunCommand executes the given command and arguments under the system'
// default shell. Really only tested under CMD and ksh
// If command fails, returns the system error and the output of the command
func RunCommand(command string, args ...string) error {
	var ret []string
	var finalargs []string
	shell, ok := os.LookupEnv("COMSPEC")
	if ok {
		finalargs = append([]string{"/C"}, command)
		finalargs = append(finalargs, args...)
		finalargs = append(finalargs, "2>&1")
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
		return fmt.Errorf("Failed to attach stdout: %v", err)
	}
	cmd.Start()
	scanner := bufio.NewScanner(outp)
	for scanner.Scan() {
		line := scanner.Text()
		ret = append(ret, line)
	}
	err = cmd.Wait()
	if err != nil {
		//fmt.Fprintf(os.Stderr, "ERROR: %s: %s\n", err, strings.Join(ret, "\n"))
		return fmt.Errorf("%s: %s", err, strings.Join(ret, "\n"))
	}
	return nil
}

// CommandCopy copies a given file or folder into the target folder
// Does not verify that the target folder exists nor if
// it is in fact a folder
// Fails if the target is the root folder
// If command fails, returns the system error and the output of the command
func CommandCopy(src string, dst string) error {
	dst = filepath.Clean(dst)
	if dst[len(dst)-1] == os.PathSeparator {
		return fmt.Errorf("Copy to root folder %s not allowed for safety", dst)
	}
	dst += string(os.PathSeparator)
	// Many safety checks to perform here...
	if runtime.GOOS == "windows" {
		// We end up using xcopy because copy will NOT handle hidden files ever
		fullDest := filepath.Join(dst, filepath.Base(src)) // Xcopy compliance, it will still ask on files
		// so we automaticall reply via echo f | xcopy ... SO UGLY
		return RunCommand("echo", "f", "|", "xcopy", "/Q", "/I", "/K", "/H", "/Y", "/R", "/S", "/E", src, fullDest)
	}
	return RunCommand("cp", "-R", src, dst)
}

// CommandMove moves a given file or folder into the target folder
// Does not verify that the target folder exists nor if
// it is in fact a folder
// Fails if the target is the root folder
// If command fails, returns the system error and the output of the command
func CommandMove(src string, dst string) error {
	dst = filepath.Clean(dst)
	if dst[len(dst)-1] == os.PathSeparator {
		return fmt.Errorf("Move to root folder %s not allowed for safety", dst)
	}
	dst += string(os.PathSeparator)
	dir := filepath.Dir(src)
	if dir[len(dir)-1] == os.PathSeparator {
		return fmt.Errorf("Moving %s from root folder not allowed for safety", dst)
	}
	// Many safety checks to perform here...
	if runtime.GOOS == "windows" {
		// hidden files will wreak havoc with move across devices
		err := RunCommand("move", "/Y", src, dst)
		if err != nil {
			// So if we get any errors we retry via copy & delete
			err = CommandCopy(src, dst)
			if err == nil {
				err = CommandDelete(src)
			}
		}
		return err
	}
	return RunCommand("mv", "-f", src, dst)
}

// CommandDelete deletes a given file or folder
// Fails if the target is the root folder
// If command fails, returns the system error and the output of the command
func CommandDelete(dst string) error {
	dst = filepath.Clean(dst)
	dir := filepath.Dir(dst)
	if dir[len(dir)-1] == os.PathSeparator {
		return fmt.Errorf("Deleting %s from root folder not allowed for safety", dst)
	}
	// Many safety checks to perform here...
	if runtime.GOOS == "windows" {
		// Deleting files in directories and deleting directories are two
		// separate things :(
		err := RunCommand("del", "/Q", "/A", dst)
		if err == nil {
			// Must ignore the error because dst may have been fully deleted by prev command
			// UGH
			/*err = */
			RunCommand("rd", "/S", "/Q", dst)
		}
		return err
	}
	return RunCommand("rm", "-rf", dst)
}
