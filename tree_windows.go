//go:build windows

package gameproc

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"

	"github.com/Rethunk-Tech/go-desktopenv/nowindow"
	"golang.org/x/sys/windows"
)

// jobs maps a started process's id to the job object holding it and its descendants.
var (
	jobsMu sync.Mutex
	jobs   = map[int]windows.Handle{}
)

// PrepareTree makes cmd start suspended, so TrackTree can put it in a job before it spawns anything that would
// escape, and without a console window when noConsole. It never sets HideWindow and keeps one the caller set:
// Windows applies HideWindow to the first window the process shows, so a program that starts the real one in a
// second process (a mod loader that runs the game) would open that window hidden. CREATE_NO_WINDOW only withholds
// the console. An existing SysProcAttr is kept.
func PrepareTree(cmd *exec.Cmd, noConsole bool) {
	if noConsole {
		nowindow.Set(cmd)
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= windows.CREATE_SUSPENDED
}

// TrackTree is called with the pid of a cmd that PrepareTree set, after Start. It puts the process in a job of its
// own, so KillTree reaches what it spawns, and resumes it. Closing the job handle later does not end the process: a
// started program may outlive its caller. An error matching ErrNotTracked, joined with its cause, means the process
// was resumed but no job holds it. Any other error means it could not be resumed; end it with KillTree.
func TrackTree(pid int) error {
	id, err := pidUint32(pid)
	if err != nil {
		return err
	}
	jobErr := trackPID(pid, id)
	if err := resume(id); err != nil {
		return err
	}
	if jobErr != nil {
		return errors.Join(ErrNotTracked, jobErr)
	}
	return nil
}

func trackPID(pid int, id uint32) error {
	h, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, id)
	if err != nil {
		return err
	}
	defer func() { _ = windows.CloseHandle(h) }()
	return trackHandle(pid, h)
}

// trackHandle puts the process behind h into a job of its own and records the job under pid. The handle needs
// PROCESS_SET_QUOTA and PROCESS_TERMINATE.
func trackHandle(pid int, h windows.Handle) error {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return err
	}
	if err := windows.AssignProcessToJobObject(job, h); err != nil {
		_ = windows.CloseHandle(job)
		return err
	}
	jobsMu.Lock()
	jobs[pid] = job
	jobsMu.Unlock()
	return nil
}

// UntrackTree releases the job once the process has exited; call it after Wait returns. A job left behind for a pid
// the system later reuses would make Terminate end that job instead of the new process.
func UntrackTree(pid int) {
	jobsMu.Lock()
	job, ok := jobs[pid]
	delete(jobs, pid)
	jobsMu.Unlock()
	if ok {
		_ = windows.CloseHandle(job)
	}
}

// terminateTracked ends the job holding pid and everything it started, and reports whether a job did.
func terminateTracked(pid int) bool {
	jobsMu.Lock()
	job, ok := jobs[pid]
	jobsMu.Unlock()
	return ok && windows.TerminateJobObject(job, 1) == nil
}

// KillTree ends the process and, when it was tracked, everything it started.
func KillTree(pid int) {
	if pid <= 0 {
		return
	}
	jobsMu.Lock()
	job, ok := jobs[pid]
	jobsMu.Unlock()
	if ok && windows.TerminateJobObject(job, 1) == nil {
		return
	}
	if p, err := os.FindProcess(pid); err == nil {
		_ = p.Kill()
	}
}

// resume resumes the one thread of a process PrepareTree started suspended.
func resume(id uint32) error {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPTHREAD, 0)
	if err != nil {
		return err
	}
	defer func() { _ = windows.CloseHandle(snap) }()
	entry := windows.ThreadEntry32{Size: uint32(unsafe.Sizeof(windows.ThreadEntry32{}))}
	for err = windows.Thread32First(snap, &entry); err == nil; err = windows.Thread32Next(snap, &entry) {
		if entry.OwnerProcessID != id {
			continue
		}
		th, err := windows.OpenThread(windows.THREAD_SUSPEND_RESUME, false, entry.ThreadID)
		if err != nil {
			return err
		}
		defer func() { _ = windows.CloseHandle(th) }()
		_, err = windows.ResumeThread(th)
		return err
	}
	return fmt.Errorf("no thread to resume in process %d", id)
}
