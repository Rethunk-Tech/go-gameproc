package gameproc

import (
	"testing"
	"time"
)

// Unix kill(0, ...) signals the caller's own group and kill(-1, ...) every process the user owns; reaching here
// after these calls means the test process survived them.
func TestPidsNoProcessHasAreRefused(t *testing.T) {
	pids := []int{0, -1}
	if big := 1; big<<32 != 0 { // a pid_t holds 31 bits; the syscall would cut this one to 0
		pids = append(pids, big<<32)
	}
	for _, pid := range pids {
		if Alive(pid) {
			t.Errorf("Alive(%d)", pid)
		}
		if err := Terminate(pid, time.Millisecond); err == nil {
			t.Errorf("Terminate(%d) succeeded", pid)
		}
		if _, err := WaitPID(pid); err == nil {
			t.Errorf("WaitPID(%d) succeeded", pid)
		}
		if _, err := ImagePath(pid); err == nil {
			t.Errorf("ImagePath(%d) succeeded", pid)
		}
		if err := TrackTree(pid); err == nil {
			t.Errorf("TrackTree(%d) succeeded", pid)
		}
		KillTree(pid)
		UntrackTree(pid)
	}
	KillTree(1)
}
