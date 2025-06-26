package cmd

import (
	"fmt"
	"os/exec"
)

// Notify sends a macOS banner via osascript.
// You can swap in terminal-notifier or beeep as needed.
func Notify(title, msg string) error {
	script := fmt.Sprintf(`display notification %q with title %q`, msg, title)
	return exec.Command("osascript", "-e", script).Run()
}
