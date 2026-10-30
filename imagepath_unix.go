//go:build !windows

package gameproc

import (
	"os"
	"path/filepath"
	"strconv"
)

// ImagePath is the executable of pid, read from /proc/<pid>/exe. It is an error where there is no /proc (macOS) and
// for a process the caller may not inspect. A deleted executable keeps the " (deleted)" suffix the kernel adds.
func ImagePath(pid int) (string, error) {
	if pid <= 0 {
		return "", errPID(pid)
	}
	return os.Readlink(filepath.Join("/proc", strconv.Itoa(pid), "exe"))
}
