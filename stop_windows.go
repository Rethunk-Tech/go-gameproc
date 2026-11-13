//go:build windows

package gameproc

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"golang.org/x/sys/windows"
)

func pidUint32(pid int) (uint32, error) {
	if pid <= 0 {
		return 0, errPID(pid)
	}
	if uint64(pid) > math.MaxUint32 {
		return 0, fmt.Errorf("pid %d out of range", pid)
	}
	return uint32(pid), nil
}

func waitMillis(d time.Duration) uint32 {
	ms := d.Milliseconds()
	if ms < 0 {
		return 0
	}
	if ms > int64(math.MaxUint32) {
		return math.MaxUint32
	}
	return uint32(ms)
}

// Stop is Terminate: Windows has no polite close for a process the caller does not own the windows of.
func Stop(_ context.Context, p Process, grace time.Duration) error {
	if !sameProcess(p) {
		return nil // The listed process is gone and its pid belongs to another.
	}
	return Terminate(p.PID, grace)
}

// sameProcess reports whether the process now holding p.PID is the one p was listed as. A Process with no Start
// cannot be told apart, so it is taken as the same.
func sameProcess(p Process) bool {
	if p.Start.IsZero() {
		return true
	}
	id, err := pidUint32(p.PID)
	if err != nil {
		return false
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, id)
	if err != nil {
		return false
	}
	defer func() { _ = windows.CloseHandle(h) }()
	now := processStart(h)
	return !now.IsZero() && now.Sub(p.Start).Abs() <= 2*time.Second
}

// Terminate ends the process, or its whole tracked tree, at once, since Windows has no polite signal for a GUI
// program, and waits up to grace for it to be gone: TerminateProcess returns before the program's files are released.
func Terminate(pid int, grace time.Duration) error {
	id, err := pidUint32(pid)
	if err != nil {
		return err
	}
	h, err := windows.OpenProcess(windows.PROCESS_TERMINATE|windows.SYNCHRONIZE, false, id)
	if err != nil {
		return alreadyExiting(pid, err, grace)
	}
	defer func() { _ = windows.CloseHandle(h) }()
	if !terminateTracked(pid) {
		if err := windows.TerminateProcess(h, 1); err != nil {
			return alreadyExiting(pid, err, grace)
		}
	}
	ev, err := windows.WaitForSingleObject(h, waitMillis(grace))
	if err != nil {
		return err
	}
	if ev != windows.WAIT_OBJECT_0 {
		return fmt.Errorf("process %d is still exiting after %s", pid, grace)
	}
	return nil
}

// alreadyExiting settles a refused kill: Windows answers access denied for a process that is already terminating (a
// crashed program still writing its report), and invalid parameter for one that is gone. Either is stopped once the
// process is gone, so only a process that outlives grace is an error.
func alreadyExiting(pid int, err error, grace time.Duration) error {
	if !errors.Is(err, windows.ERROR_ACCESS_DENIED) && !errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
		return err
	}
	if exitedWithin(func() bool { return !Alive(pid) }, grace) {
		return nil
	}
	return fmt.Errorf("process %d is still running after %s: %w", pid, grace, err)
}

// Alive reports whether a process with this pid is still running.
func Alive(pid int) bool {
	id, err := pidUint32(pid)
	if err != nil {
		return false
	}
	h, err := windows.OpenProcess(windows.SYNCHRONIZE, false, id)
	if err != nil {
		// An elevated or protected process refuses the handle yet runs; only "no such pid" means gone.
		return errors.Is(err, windows.ERROR_ACCESS_DENIED)
	}
	defer func() { _ = windows.CloseHandle(h) }()
	// The handle is signalled once the process ends; a program may itself exit with code 259, which
	// GetExitCodeProcess reports the same as "still active".
	ev, err := windows.WaitForSingleObject(h, 0)
	return err == nil && ev == uint32(windows.WAIT_TIMEOUT)
}
