package gameproc

import "errors"

// Status is how a process ended. Signal is the Unix signal name ("SIGSEGV") of a process a signal killed, and is
// empty on Windows. A killed process has Code 128 plus the signal number.
type Status struct {
	Code   int
	Signal string
}

// ErrNotTracked is TrackTree's error, joined with its cause, for a process that was resumed but that no job holds:
// KillTree then ends only the process.
var ErrNotTracked = errors.New("gameproc: the process runs outside a job")

// ErrNotChild is WaitPID's error, joined with the wait error, for a Unix process that is not the caller's child: it
// was waited out, but its exit status is unknown.
var ErrNotChild = errors.New("gameproc: the process is not a child, so its exit status is unknown")

// ErrNoConsole is StartOwnConsole's error off Windows.
var ErrNoConsole = errors.New("gameproc: a program with its own console starts only on Windows")
