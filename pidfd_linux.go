//go:build linux

package gameproc

import (
	"errors"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// pidfdTerminate is Terminate through a pidfd, which names the process that held pid when it was opened and never a
// later one that is given the number. p.Start, when known, is checked after the open, so a pid recycled since the
// listing is refused. handled is false when the kernel has no pidfd (before 5.3) or refuses one, and the caller
// signals the bare pid then.
func pidfdTerminate(p Process, grace time.Duration) (handled bool, err error) {
	fd, err := unix.PidfdOpen(p.PID, 0)
	if err != nil {
		if errors.Is(err, syscall.ESRCH) {
			return true, nil
		}
		return false, nil
	}
	defer func() { _ = unix.Close(fd) }()
	if !sameProcess(p) {
		return true, nil
	}
	if err := unix.PidfdSendSignal(fd, unix.SIGTERM, nil, 0); err != nil {
		if errors.Is(err, syscall.ESRCH) {
			return true, nil
		}
		return true, err
	}
	if pidfdExited(fd, grace) {
		return true, nil
	}
	if err := unix.PidfdSendSignal(fd, unix.SIGKILL, nil, 0); err != nil && !errors.Is(err, syscall.ESRCH) {
		return true, err
	}
	return true, nil
}

// pidfdExited waits up to grace for the process behind fd to terminate, which makes the pidfd readable.
func pidfdExited(fd int, grace time.Duration) bool {
	deadline := time.Now().Add(grace)
	for {
		left := time.Until(deadline)
		if left <= 0 {
			return false
		}
		fds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}} //nolint:gosec // G115: a descriptor fits int32
		n, err := unix.Poll(fds, int(left.Milliseconds())+1)
		if errors.Is(err, syscall.EINTR) {
			continue
		}
		return err == nil && n > 0
	}
}

// pidfdWait blocks until the process behind pid exits. ok is false when no pidfd could be had.
func pidfdWait(pid int, start time.Time) (ok bool) {
	fd, err := unix.PidfdOpen(pid, 0)
	if err != nil {
		return errors.Is(err, syscall.ESRCH)
	}
	defer func() { _ = unix.Close(fd) }()
	if !sameProcess(Process{PID: pid, Start: start}) {
		return true
	}
	for {
		fds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}} //nolint:gosec // G115: a descriptor fits int32
		n, err := unix.Poll(fds, -1)
		if errors.Is(err, syscall.EINTR) {
			continue
		}
		return err == nil && n > 0
	}
}
