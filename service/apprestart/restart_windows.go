//go:build windows

package apprestart

import "fmt"

// Windows deployments require an external process manager. Linux is the
// supported server platform for the in-process re-exec button.
func Schedule() error {
	return fmt.Errorf("manual in-process restart is supported on Linux and macOS only")
}
