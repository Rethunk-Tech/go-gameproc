//go:build windows

package gameproc

import (
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Processes lists processes whose image name is one of names (in any case, with or without ".exe") through Toolhelp.
// procDir is ignored. Command lines are not read (that needs the process's PEB), so Args stays nil. A program in
// the Windows folder is skipped; one whose image path cannot be read is kept with an empty Exe.
func Processes(_ string, names ...string) ([]Process, error) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer func() { _ = windows.CloseHandle(snap) }()
	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	var out []Process
	for err = windows.Process32First(snap, &entry); err == nil; err = windows.Process32Next(snap, &entry) {
		exe := trimExe(windows.UTF16ToString(entry.ExeFile[:]))
		if slices.ContainsFunc(names, func(name string) bool { return strings.EqualFold(exe, name) }) {
			if path, start := imagePath(entry.ProcessID); !systemProgram(path) {
				out = append(out, Process{PID: int(entry.ProcessID), Exe: path, Start: start})
			}
		}
	}
	return out, nil
}

// Named reports whether p is one of names, as Processes matched it. A process whose image path was unreadable may be
// any of them.
func (p Process) Named(names ...string) bool {
	exe := trimExe(filepath.Base(p.Exe))
	return p.Exe == "" || slices.ContainsFunc(names, func(name string) bool { return strings.EqualFold(exe, name) })
}

// imagePath is the executable path and creation time of pid, or empty when the process cannot be opened.
func imagePath(pid uint32) (string, time.Time) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return "", time.Time{}
	}
	defer func() { _ = windows.CloseHandle(h) }()
	var buf [windows.MAX_LONG_PATH]uint16
	n := uint32(windows.MAX_LONG_PATH)
	path := ""
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &n); err == nil {
		path = windows.UTF16ToString(buf[:n])
	}
	return path, processStart(h)
}

func processStart(h windows.Handle) time.Time {
	var creation, exit, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(h, &creation, &exit, &kernel, &user); err != nil {
		return time.Time{}
	}
	return time.Unix(0, creation.Nanoseconds())
}

// ImagePath is the executable path of pid (QueryFullProcessImageName).
func ImagePath(pid int) (string, error) {
	id, err := pidUint32(pid)
	if err != nil {
		return "", err
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, id)
	if err != nil {
		return "", err
	}
	defer func() { _ = windows.CloseHandle(h) }()
	var buf [windows.MAX_LONG_PATH]uint16
	n := uint32(windows.MAX_LONG_PATH)
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &n); err != nil {
		return "", err
	}
	return windows.UTF16ToString(buf[:n]), nil
}
