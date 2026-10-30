package gameproc

import (
	"runtime"
	"testing"
	"time"
)

func TestKillTreeEndsWhatTheProgramStarted(t *testing.T) {
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
	KillTree(pid)
	await(t, done)
	waitGone(t, grandchild)
}

func TestTerminateEndsTheTrackedTreeOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Unix Terminate signals the process, not its group")
	}
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
	if err := Terminate(pid, 10*time.Second); err != nil {
		t.Fatal(err)
	}
	await(t, done)
	waitGone(t, grandchild)
}
