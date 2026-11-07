//go:build linux

package gameproc

import (
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"
)

// listed starts a sleeping helper and returns it as Processes would list it.
func listed(t *testing.T) Process {
	t.Helper()
	cmd := helperCmd(t, "sleep")
	startHelper(t, cmd)
	reap(cmd)
	pid := cmd.Process.Pid
	start := startTime("/proc", strconv.Itoa(pid))
	if start.IsZero() {
		t.Fatal("no start time")
	}
	return Process{PID: pid, Start: start}
}

// A pid recycled since the listing belongs to another process: Stop leaves it alone.
func TestStopLeavesAProcessThatIsNotTheListedOne(t *testing.T) {
	p := listed(t)
	p.Start = p.Start.Add(-time.Hour)
	if err := Stop(t.Context(), p, time.Second); err != nil {
		t.Fatal(err)
	}
	if !Alive(p.PID) {
		t.Fatal("Stop killed a process that was not the listed one")
	}
	p.Start = p.Start.Add(time.Hour)
	if err := Stop(t.Context(), p, 10*time.Second); err != nil {
		t.Fatal(err)
	}
	waitGone(t, p.PID)
}

func TestSameProcessComparesStartTimes(t *testing.T) {
	if !sameProcess(Process{PID: 1}) {
		t.Fatal("a Process with no start is always the same")
	}
	p := listed(t)
	if !sameProcess(p) {
		t.Fatal("the listed process is not itself")
	}
	p.Start = p.Start.Add(time.Minute)
	if sameProcess(p) {
		t.Fatal("another start time counted as the same process")
	}
}

// KillTree after the leader's pid went to another process must not signal that process's group.
func TestKillTreeSparesARecycledLeader(t *testing.T) {
	cmd := helperCmd(t, "sleep")
	PrepareTree(cmd, true)
	startHelper(t, cmd)
	done := reap(cmd)
	pid := cmd.Process.Pid
	if err := TrackTree(pid); err != nil {
		t.Fatal(err)
	}
	treeLeaders.Lock()
	truth := treeLeaders.start[pid]
	treeLeaders.start[pid] = truth.Add(-time.Hour)
	treeLeaders.Unlock()
	KillTree(pid)
	time.Sleep(200 * time.Millisecond)
	if !Alive(pid) {
		t.Fatal("KillTree signalled a group whose leader is not the tracked one")
	}
	treeLeaders.Lock()
	treeLeaders.start[pid] = truth
	treeLeaders.Unlock()
	KillTree(pid)
	await(t, done)
}

// localHost stands in for flatpak-spawn by running the host command on this machine, so the real identity script runs.
func localHost(t *testing.T) {
	t.Helper()
	t.Setenv("FLATPAK_ID", "x")
	old := hostOutput
	t.Cleanup(func() { hostOutput = old })
	hostOutput = func(name string, args ...string) ([]byte, error) {
		return exec.CommandContext(t.Context(), name, args...).Output() //nolint:gosec // G204: the test runs the host script on this machine
	}
}

func TestFlatpakStopChecksTheHostStartTimeInTheSameCommand(t *testing.T) {
	cmd := helperCmd(t, "sleep")
	startHelper(t, cmd)
	done := reap(cmd)
	pid := cmd.Process.Pid
	info, err := os.Stat("/proc/" + strconv.Itoa(pid))
	if err != nil {
		t.Fatal(err)
	}
	localHost(t)
	other := Process{PID: pid, Start: info.ModTime().Add(-time.Hour)}
	if err := Stop(t.Context(), other, 500*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if !Alive(pid) {
		t.Fatal("a process created at another time was signalled")
	}
	if err := Stop(t.Context(), Process{PID: pid, Start: info.ModTime()}, 10*time.Second); err != nil {
		t.Fatal(err)
	}
	await(t, done)
}
