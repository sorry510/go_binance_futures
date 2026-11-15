//go:build linux || darwin

// Package apprestart supports a manually triggered in-process re-exec.
// It does not invoke PM2, a shell, or a separate supervisor.
package apprestart

import (
	"fmt"
	"os"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/beego/beego/v2/core/logs"
)

var restartScheduled atomic.Bool

// Schedule validates the running executable before acknowledging the request.
// The delay gives the HTTP handler time to flush its response to the browser.
func Schedule() error {
	if !restartScheduled.CompareAndSwap(false, true) {
		return fmt.Errorf("application restart already requested")
	}
	exe, err := os.Executable()
	if err != nil {
		restartScheduled.Store(false)
		return fmt.Errorf("locate current executable: %w", err)
	}
	info, err := os.Stat(exe)
	if err != nil || !info.Mode().IsRegular() {
		restartScheduled.Store(false)
		return fmt.Errorf("current executable is not available")
	}
	go func() {
		time.Sleep(1200 * time.Millisecond)
		logs.Warning("manually requested application restart: re-executing the Go binary")
		logs.GetBeeLogger().Flush()
		if err := syscall.Exec(exe, os.Args, os.Environ()); err != nil {
			logs.Error("application re-exec failed: ", err)
			restartScheduled.Store(false)
		}
	}()
	return nil
}
