//go:build !windows

package gameproc

import (
	"errors"
	"os"
	"strconv"
	"syscall"
	"time"

	"github.com/Rethunk-Tech/go-desktopenv/flatpak"
)

// WaitPID waits until pid exits. The exit status is readable only for the caller's own child
// (os.FindProcess(pid).Wait()). For any other process it polls Alive until the exit and returns an empty Status and an
// error matching ErrNotChild, so a caller can tell an unknown exit from a clean one. Inside a Flatpak it returns a nil
// error once the host probe says the process is gone.
func WaitPID(pid int) (Status, error) {
	if !validPID(pid) {
		return Status{}, errPID(pid)
	}
	if flatpak.Inside() {
		// Host PIDs mean nothing in the sandbox's PID namespace. Bind the start on the first
		// alive probe so a later holder of the number reads as gone.
		var start time.Time
		for {
			alive, err := hostProbe(pid, start)
			if err != nil {
				return Status{}, err
			}
			if !alive {
				return Status{}, nil
			}
			if start.IsZero() {
				start, err = hostStart(pid)
				if err != nil {
					return Status{}, err
				}
				continue
			}
			time.Sleep(time.Second)
		}
	}
	proc, err := os.FindProcess(pid)
	if err == nil {
		state, werr := proc.Wait()
		if werr == nil {
			if ws, ok := state.Sys().(syscall.WaitStatus); ok {
				return waitStatus(ws), nil
			}
			return Status{Code: state.ExitCode()}, nil
		}
		err = werr
	}
	// The pid may be recycled while we wait; the start time (or a pidfd) tells the process we were asked about from
	// whatever takes its number.
	start := startTime("/proc", strconv.Itoa(pid))
	if !pidfdWait(pid, start) {
		for Alive(pid) && sameProcess(Process{PID: pid, Start: start}) {
			time.Sleep(100 * time.Millisecond)
		}
	}
	if errors.Is(err, syscall.ECHILD) {
		err = errors.Join(ErrNotChild, err)
	}
	return Status{}, err
}
