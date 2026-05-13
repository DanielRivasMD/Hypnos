/*
Copyright © 2026 Daniel Rivas <danielrivasmd@gmail.com>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package cmd

////////////////////////////////////////////////////////////////////////////////////////////////////

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

////////////////////////////////////////////////////////////////////////////////////////////////////

func spawnProbe(meta *probeMeta) (int, error) {
	exe, _ := os.Executable()

	args := []string{
		"hibernate-worker",
		"--probe", meta.Probe,
		"--log", strings.TrimSuffix(filepath.Base(meta.LogPath), ".log"),
		"--script", meta.Script,
		"--duration", meta.Duration.String(),
	}

	if meta.Iterations > 0 {
		args = append(args, "--iterations", strconv.Itoa(meta.Iterations))
	} else if meta.Recurrent {
		args = append(args, "--recurrent")
	}
	if meta.Notify {
		args = append(args, "--notify-only")
	}
	if meta.Carbonite {
		args = append(args, "--carbonite")
	}

	f, err := os.OpenFile(meta.LogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return 0, err
	}

	cmd := exec.Command(exe, args...)
	cmd.Stdout = f
	cmd.Stderr = f
	if err := cmd.Start(); err != nil {
		f.Close()
		return 0, err
	}
	return cmd.Process.Pid, nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func runDowntime(d time.Duration, onDone func()) {
	time.AfterFunc(d, onDone)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func runAsDaemon(script string, logFile *os.File) error {
	if err := syscall.Dup2(int(logFile.Fd()), 1); err != nil {
		return fmt.Errorf("dup2 stdout: %w", err)
	}
	if err := syscall.Dup2(int(logFile.Fd()), 2); err != nil {
		return fmt.Errorf("dup2 stderr: %w", err)
	}
	_ = logFile.Close()

	argv := []string{"/bin/sh", "-c", script}
	env := os.Environ()
	return syscall.Exec(argv[0], argv, env)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func notify(title, msg string) error {
	if tnPath, err := exec.LookPath("terminal-notifier"); err == nil {
		cmd := exec.Command(
			tnPath,
			"-title", title,
			"-message", msg,
			"-sender", "com.apple.Terminal",
		)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("terminal-notifier error: %v – %s", err, output)
		}
		return nil
	}

	if osaPath, err := exec.LookPath("osascript"); err == nil {
		script := fmt.Sprintf(`display notification %q with title %q`, msg, title)
		cmd := exec.Command(osaPath, "-e", script)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("osascript error: %v – %s", err, output)
		}
		return nil
	}

	return fmt.Errorf("no macOS notifier found: install terminal-notifier or ensure osascript is in PATH")
}

////////////////////////////////////////////////////////////////////////////////////////////////////
