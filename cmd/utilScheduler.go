package cmd

import "time"

// RunDowntime sleeps for d, then calls onDone.
func RunDowntime(d time.Duration, onDone func()) {
	go func() {
		time.Sleep(d)
		onDone()
	}()
}
