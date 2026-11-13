//go:build linux

package gameproc

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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

// startedAs runs a sleeping process under the given file name; name holds the awkward bytes a test asks for, such as
// ") " and "(", which shift naive stat parsing, or a newline. The sleeper is a copy of this test binary, which sleeps
// under any name it is copied to: /bin/sleep on Ubuntu 26.04 is one multicall coreutils that exits on a name it does
// not know.
func startedAs(t *testing.T, name string) (pid int, done <-chan error) {
	t.Helper()
	exe := filepath.Join(t.TempDir(), name)
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(self) //nolint:gosec // G304: self is this test binary, from os.Executable
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exe, b, 0o700); err != nil { //nolint:gosec // G306: an executable copy in the test's own folder
		t.Fatal(err)
	}
	cmd := exec.CommandContext(t.Context(), exe, "-test.run=^$") //nolint:gosec // G204: a copy of this test binary made by the test
	cmd.Env = append(os.Environ(), helperEnv+"=sleep")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill() })
	return cmd.Process.Pid, reap(cmd)
}

// The host listing and the host identity check read the same start as startTime does, comm with ") " included, and
// one holding a newline, which makes stat span lines where a first-line read would stop early.
func TestHostStartMatchesStartTimeForAnAwkwardCommandName(t *testing.T) {
	for _, name := range []string{"x) y (z", "x\n) y (z"} {
		t.Run(strings.ReplaceAll(name, "\n", `\n`), func(t *testing.T) {
			pid, _ := startedAs(t, name)
			want := startTime("/proc", strconv.Itoa(pid))
			if want.IsZero() {
				t.Fatal("no start time")
			}
			out, err := exec.CommandContext(t.Context(), "sh", "-c", hostScript).Output()
			if err != nil {
				t.Fatal(err)
			}
			listing, _, _ := strings.Cut(string(out), "\x00")
			mark := "\n" + strconv.Itoa(pid) + " " + strconv.FormatInt(want.Unix(), 10)
			if !strings.Contains("\n"+listing, mark+"\n") && !strings.Contains("\n"+listing, mark+" ") {
				t.Fatalf("host listing has no %d %d line", pid, want.Unix())
			}
		})
	}
}

func TestFlatpakStopChecksTheHostStartTimeInTheSameCommand(t *testing.T) {
	for _, name := range []string{"x) y (z", "x\n) y (z"} {
		t.Run(strings.ReplaceAll(name, "\n", `\n`), func(t *testing.T) {
			pid, done := startedAs(t, name)
			start := startTime("/proc", strconv.Itoa(pid))
			localHost(t)
			recycled := Process{PID: pid, Start: start.Add(-time.Hour)}
			if err := Stop(t.Context(), recycled, 500*time.Millisecond); err != nil {
				t.Fatal(err)
			}
			if !Alive(pid) {
				t.Fatal("a process started at another time was signalled")
			}
			if err := Stop(t.Context(), Process{PID: pid, Start: start}, 10*time.Second); err != nil {
				t.Fatal(err)
			}
			await(t, done)
		})
	}
}
