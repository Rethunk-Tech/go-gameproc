package gameproc

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

// exitCode is a process exit status for a start that did not go through os/exec.
type exitCode int

func (e exitCode) Error() string { return fmt.Sprintf("exit status %d", int(e)) }

// ExitCode is read by StatusOf.
func (e exitCode) ExitCode() int { return int(e) }

// StartOwnConsole starts a program in a new console that it reads and writes itself, as a desktop shell or a game
// launcher starts a console program. os/exec cannot: it always passes standard handles (STARTF_USESTDHANDLES), and
// when none are set they are NUL, so the console stays empty and an input loop reads end-of-file. Redirecting a
// program that has a console also breaks the console calls it makes (colours, cursor, window size).
//
// env is the complete environment of the program; empty means the caller's own. The process is tracked as TrackTree
// tracks one (it runs untracked when no job can hold it) and is released when it exits. The channel receives the
// exit: nil for code 0, else an error with ExitCode(), which StatusOf maps. It takes no context: the program
// outlives the caller's request.
func StartOwnConsole(env []string, dir, name string, args []string) (<-chan error, error) {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, syscall.EscapeArg(name))
	for _, a := range args {
		parts = append(parts, syscall.EscapeArg(a))
	}
	cmdLine, err := windows.UTF16PtrFromString(strings.Join(parts, " "))
	if err != nil {
		return nil, err
	}
	app, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}
	var cwd *uint16
	if dir != "" {
		if cwd, err = windows.UTF16PtrFromString(dir); err != nil {
			return nil, err
		}
	}
	flags := uint32(windows.CREATE_NEW_CONSOLE | windows.CREATE_SUSPENDED)
	var block *uint16
	if len(env) > 0 {
		flags |= windows.CREATE_UNICODE_ENVIRONMENT
		block = envBlock(env)
	}
	si := windows.StartupInfo{}
	si.Cb = uint32(unsafe.Sizeof(si))
	var pi windows.ProcessInformation
	if err := windows.CreateProcess(app, cmdLine, nil, nil, false, flags, block, cwd, &si, &pi); err != nil {
		return nil, &os.PathError{Op: "start", Path: name, Err: err}
	}
	pid := int(pi.ProcessId)
	_ = trackHandle(pid, pi.Process)
	_, resumeErr := windows.ResumeThread(pi.Thread)
	_ = windows.CloseHandle(pi.Thread)
	if resumeErr != nil {
		KillTree(pid)
		UntrackTree(pid)
		_ = windows.CloseHandle(pi.Process)
		return nil, &os.PathError{Op: "start", Path: name, Err: resumeErr}
	}
	done := make(chan error, 1)
	go func() {
		defer func() { _ = windows.CloseHandle(pi.Process) }()
		defer UntrackTree(pid)
		if _, err := windows.WaitForSingleObject(pi.Process, windows.INFINITE); err != nil {
			done <- err
			return
		}
		var code uint32
		if err := windows.GetExitCodeProcess(pi.Process, &code); err != nil {
			done <- err
			return
		}
		if code == 0 {
			done <- nil
			return
		}
		done <- exitCode(code)
	}()
	return done, nil
}

// envBlock is env as a sorted, NUL-separated, double-NUL-terminated UTF-16 block, as CreateProcess wants it; a
// later duplicate of a variable wins, as with os/exec.
func envBlock(env []string) *uint16 {
	out := envUTF16(env)
	return &out[0]
}

func envUTF16(env []string) []uint16 {
	last := map[string]string{}
	for _, kv := range env {
		k, _, _ := strings.Cut(kv, "=")
		last[strings.ToUpper(k)] = kv
	}
	keys := slices.Sorted(maps.Keys(last))
	var out []uint16
	for _, k := range keys {
		out = append(out, utf16.Encode([]rune(last[k]))...)
		out = append(out, 0)
	}
	return append(out, 0)
}
