//go:build windows

package gameproc

import (
	"errors"

	"golang.org/x/sys/windows"
)

// StatusOf maps exec.Cmd.Wait's error (or a StartOwnConsole result) to a Status. Signal is always empty.
func StatusOf(err error) Status {
	if err == nil {
		return Status{}
	}
	if ee, ok := errors.AsType[interface {
		error
		ExitCode() int
	}](err); ok {
		return Status{Code: ee.ExitCode()}
	}
	return Status{Code: 1}
}

// WaitPID waits until pid exits and reports its exit code, for a child or not.
func WaitPID(pid int) (Status, error) {
	id, err := pidUint32(pid)
	if err != nil {
		return Status{}, err
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, id)
	if err != nil {
		return Status{}, err
	}
	defer func() { _ = windows.CloseHandle(h) }()
	if _, err := windows.WaitForSingleObject(h, windows.INFINITE); err != nil {
		return Status{}, err
	}
	var code uint32
	if err := windows.GetExitCodeProcess(h, &code); err != nil {
		return Status{}, err
	}
	return Status{Code: int(code)}, nil
}
