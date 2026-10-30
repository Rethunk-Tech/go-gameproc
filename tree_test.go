package gameproc

import (
	"testing"
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
