//go:build !windows

package gameproc

import (
	"errors"
	"os"
	"syscall"
	"time"

	"github.com/Rethunk-Tech/go-desktopenv/flatpak"
)

// WaitPID waits until pid exits. The exit status is readable only for the caller's own child
// (os.FindProcess(pid).Wait()). For any other process it polls Alive until the exit and returns an empty Status and an
// error matching ErrNotChild, so a caller can tell an unknown exit from a clean one. Inside a Flatpak it returns a nil
// error once the host probe says the process is gone.
func WaitPID(pid int) (Status, error) {
	if pid <= 0 {
		return Status{}, errPID(pid)
	}
	if flatpak.Inside() {
		// Host PIDs mean nothing in the sandbox's PID namespace.
		for {
			alive, err := hostProbe(pid)
			if err != nil {
				return Status{}, err
			}
			if !alive {
				return Status{}, nil
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
	for Alive(pid) {
		time.Sleep(100 * time.Millisecond)
	}
	if errors.Is(err, syscall.ECHILD) {
		err = errors.Join(ErrNotChild, err)
	}
	return Status{}, err
}
