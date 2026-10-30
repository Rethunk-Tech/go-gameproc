package gameproc

import (
	"runtime"
	"testing"
	"time"
)

func TestEndingATrackedTreeEndsWhatTheProgramStarted(t *testing.T) {
	enders := map[string]func(t *testing.T, pid int){
		"KillTree": func(_ *testing.T, pid int) { KillTree(pid) },
	}
	// Unix Terminate signals the process, not its group; Windows ends the tracked job.
	if runtime.GOOS == "windows" {
		enders["Terminate"] = func(t *testing.T, pid int) {
			if err := Terminate(pid, 10*time.Second); err != nil {
				t.Fatal(err)
			}
		}
	}
	for name, end := range enders {
		t.Run(name, func(t *testing.T) {
			cmd := helperCmd(t, "spawn")
			PrepareTree(cmd, true)
			r := startHelper(t, cmd)
			pid := cmd.Process.Pid
			if err := TrackTree(pid); err != nil {
				t.Fatalf("the tree is not tracked: %v", err)
			}
			t.Cleanup(func() { UntrackTree(pid) })
			grandchild := readPID(t, r)
			done := reap(cmd)
			end(t, pid)
			await(t, done)
			waitGone(t, grandchild)
		})
	}
}

func TestTrackTreeRefusesNoPID(t *testing.T) {
	for _, pid := range []int{0, -3} {
		if TrackTree(pid) == nil {
			t.Errorf("pid %d was tracked", pid)
		}
	}
}
