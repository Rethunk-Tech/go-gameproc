//go:build unix

package gameproc

import (
	"errors"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

// StatusOf maps exec.Cmd.Wait's error to a Status. A process killed by a signal has Code 128 plus the signal number
// and the signal's name in Signal. An error that is not an exit is Code 1.
func StatusOf(waitErr error) Status {
	if waitErr == nil {
		return Status{}
	}
	if ee, ok := errors.AsType[*exec.ExitError](waitErr); ok {
		s := Status{Code: ee.ExitCode()}
		if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			s.Signal = signalName(ws.Signal())
			if s.Code < 0 {
				s.Code = 128 + int(ws.Signal())
			}
		}
		return s
	}
	return Status{Code: 1}
}

func waitStatus(ws syscall.WaitStatus) Status {
	if ws.Signaled() {
		sig := ws.Signal()
		return Status{Code: 128 + int(sig), Signal: signalName(sig)}
	}
	return Status{Code: ws.ExitStatus()}
}

func signalName(sig syscall.Signal) string {
	if name := unix.SignalName(sig); name != "" {
		return name
	}
	return strings.TrimSpace(sig.String())
}
