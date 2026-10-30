//go:build unix

package gameproc

import (
	"syscall"
	"testing"
)

func TestWaitStatus(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		ws   syscall.WaitStatus
		want Status
	}{
		{"exit 3", 3 << 8, Status{Code: 3}},
		{"clean exit", 0, Status{}},
		{"SIGKILL", syscall.WaitStatus(syscall.SIGKILL), Status{Code: 137, Signal: "SIGKILL"}},
		{"SIGSEGV", syscall.WaitStatus(syscall.SIGSEGV), Status{Code: 139, Signal: "SIGSEGV"}},
	} {
		if got := waitStatus(tc.ws); got != tc.want {
			t.Errorf("%s: %+v, want %+v", tc.name, got, tc.want)
		}
	}
}
