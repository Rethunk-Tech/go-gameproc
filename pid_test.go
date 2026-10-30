package gameproc

import (
	"testing"
	"time"
)

// Unix kill(0, ...) signals the caller's own group and kill(-1, ...) every process the user owns; reaching here
// after these calls means the test process survived them.
func TestPidsNoProcessHasAreRefused(t *testing.T) {
	for _, pid := range []int{0, -1} {
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
