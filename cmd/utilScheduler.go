package cmd

import "time"

// RunDowntime waits for d, then invokes onDone in its own goroutine,
// without creating a sleeping goroutine up front.
func RunDowntime(d time.Duration, onDone func()) {
	time.AfterFunc(d, onDone)
}
