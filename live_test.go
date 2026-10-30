package gameproc

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
)

func TestProcessesFindsARunningProgram(t *testing.T) {
	cmd := helperCmd(t, "sleep")
	startHelper(t, cmd)
	name := trimExe(filepath.Base(os.Args[0]))
	procDir := "/proc"
	if runtime.GOOS == "windows" {
		procDir = ""
	}
	procs, err := Processes(procDir, name)
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(procs, func(p Process) bool { return p.PID == cmd.Process.Pid })
	if i < 0 {
		t.Fatalf("pid %d not in %+v", cmd.Process.Pid, procs)
	}
	p := procs[i]
	if !p.Named(name) || !p.ExeIs(name) {
		t.Errorf("%+v is not %q", p, name)
	}
	if runtime.GOOS == "windows" {
		if p.Args != nil || !safefs.SamePath(p.Exe, os.Args[0]) {
			t.Errorf("windows entry %+v", p)
		}
		return
	}
	dir, err := safefs.EvalSymlinks(filepath.Dir(os.Args[0]))
	if err != nil {
		t.Fatal(err)
	}
	if !p.RunsFrom(dir) {
		t.Errorf("%+v does not run from %s", p, dir)
	}
	if p.Start.IsZero() || time.Since(p.Start) > time.Minute {
		t.Errorf("start %v", p.Start)
	}
	if got, err := ImagePath(p.PID); err != nil || got != p.Exe {
		t.Errorf("ImagePath = %q, %v; Exe = %q", got, err, p.Exe)
	}
}

func TestImagePathOfARunningProgram(t *testing.T) {
	cmd := helperCmd(t, "sleep")
	startHelper(t, cmd)
	got, err := ImagePath(cmd.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	if want, _ := safefs.EvalSymlinks(os.Args[0]); !safefs.SamePath(got, want) {
		t.Fatalf("ImagePath = %q, want %q", got, want)
	}
}

func TestTerminateEndsAProgram(t *testing.T) {
	cmd := helperCmd(t, "sleep")
	startHelper(t, cmd)
	done := reap(cmd)
	if err := Terminate(cmd.Process.Pid, 10*time.Second); err != nil {
		t.Fatal(err)
	}
	await(t, done)
	if Alive(cmd.Process.Pid) {
		t.Fatal("still alive")
	}
	if err := Terminate(cmd.Process.Pid, time.Second); err != nil {
		t.Fatalf("terminating an exited process: %v", err)
	}
}

func TestStopEndsAProgram(t *testing.T) {
	cmd := helperCmd(t, "sleep")
	startHelper(t, cmd)
	done := reap(cmd)
	if err := Stop(t.Context(), Process{PID: cmd.Process.Pid}, 10*time.Second); err != nil {
		t.Fatal(err)
	}
	await(t, done)
}

func TestWaitPIDReportsAChildsExitCode(t *testing.T) {
	cmd := helperCmd(t, "exit=3")
	startHelper(t, cmd)
	got, err := WaitPID(cmd.Process.Pid)
	if err != nil || got.Code != 3 {
		t.Fatalf("WaitPID = %+v, %v", got, err)
	}
}

func TestWaitPIDWaitsForAProcessThatIsNotAChild(t *testing.T) {
	cmd := helperCmd(t, "spawn")
	grandchild := readPID(t, startHelper(t, cmd))
	done := make(chan Status, 1)
	go func() {
		s, err := WaitPID(grandchild)
		if runtime.GOOS != "windows" && !errors.Is(err, ErrNotChild) || runtime.GOOS == "windows" && err != nil {
			t.Errorf("WaitPID: %v", err)
		}
		done <- s
	}()
	select {
	case <-done:
		t.Fatal("returned while the process runs")
	case <-time.After(300 * time.Millisecond):
	}
	proc, err := os.FindProcess(grandchild)
	if err != nil {
		t.Fatal(err)
	}
	if err := proc.Kill(); err != nil {
		t.Fatal(err)
	}
	select {
	case s := <-done:
		if runtime.GOOS != "windows" && s != (Status{}) {
			t.Errorf("a non-child has no readable status, got %+v", s)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("WaitPID did not return after the exit")
	}
}

func TestStatusOfMapsAWaitError(t *testing.T) {
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^$") //nolint:gosec // G204: re-executes the test binary
	cmd.Env = append(os.Environ(), helperEnv+"=exit=5")
	if got := StatusOf(cmd.Run()); got != (Status{Code: 5}) {
		t.Fatalf("StatusOf = %+v", got)
	}
	if got := StatusOf(nil); got != (Status{}) {
		t.Fatalf("StatusOf(nil) = %+v", got)
	}
}

func TestTermOkHelperExitsCleanlyOnTerminate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows ends a process without a polite signal")
	}
	cmd := helperCmd(t, "term-ok")
	r := startHelper(t, cmd)
	if line, err := r.ReadString('\n'); err != nil || line != "ready\n" {
		t.Fatalf("helper: %q, %v", line, err)
	}
	done := reap(cmd)
	if err := Terminate(cmd.Process.Pid, 10*time.Second); err != nil {
		t.Fatal(err)
	}
	if got := StatusOf(<-done); got != (Status{}) {
		t.Fatalf("exit %+v, want a clean exit before the grace ran out", got)
	}
}
