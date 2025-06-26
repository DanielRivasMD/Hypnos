package cmd

import (
	"fmt"
	"os/exec"
)

// Notify sends a macOS user notification via one of:
// 1) terminal-notifier (preferred, with -sender for GUI session access)
// 2) AppleScript (osascript fallback)
// Returns an error if no supported notifier is found or the command fails.
func Notify(title, msg string) error {
	// 1) Try terminal-notifier if installed
	if tnPath, err := exec.LookPath("terminal-notifier"); err == nil {
		cmd := exec.Command(
			tnPath,
			"-title", title,
			"-message", msg,
			"-sender", "com.apple.Terminal", // ensure Notification Center accepts it
		)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("terminal-notifier error: %v – %s", err, output)
		}
		return nil
	}

	// 2) Fallback to AppleScript via osascript
	if osaPath, err := exec.LookPath("osascript"); err == nil {
		// display notification "<msg>" with title "<title>"
		script := fmt.Sprintf(`display notification %q with title %q`, msg, title)
		cmd := exec.Command(osaPath, "-e", script)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("osascript error: %v – %s", err, output)
		}
		return nil
	}

	return fmt.Errorf("no macOS notifier found: install terminal-notifier or ensure osascript is in PATH")
}
