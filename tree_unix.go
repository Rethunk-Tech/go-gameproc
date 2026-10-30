//go:build !windows

package gameproc

import (
	"os/exec"
	"syscall"
)

// PrepareTree puts the process in a group of its own, so KillTree can end it with everything it started. The
// noConsole argument has no effect off Windows. Any SysProcAttr the caller set is kept.
func PrepareTree(cmd *exec.Cmd, _ bool) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

// TrackTree needs nothing off Windows: the process group is complete at the start. It still refuses a pid no process
// has, so the same call is safe on every OS.
func TrackTree(pid int) error {
	if pid <= 0 {
		return errPID(pid)
	}
	return nil
}

// KillTree sends SIGTERM to the process group PrepareTree made pid the leader of. It refuses pid 1 and below, since
// kill(-1, ...) signals every process the user owns.
func KillTree(pid int) {
	if pid <= 1 {
		return
	}
	_ = syscall.Kill(-pid, syscall.SIGTERM)
}

// UntrackTree is a no-op off Windows.
func UntrackTree(int) {}
