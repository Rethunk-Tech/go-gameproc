//go:build windows

package gameproc

import (
	"math"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestPIDUint32RejectsWhatNoProcessHas(t *testing.T) {
	t.Parallel()
	for _, pid := range []int{-1, 0} {
		if _, err := pidUint32(pid); err == nil {
			t.Fatalf("pid %d accepted", pid)
		}
	}
}

func TestWaitMillisClampsLargeDuration(t *testing.T) {
	t.Parallel()
	d := time.Duration(int64(math.MaxUint32)+1) * time.Millisecond
	if got := waitMillis(d); got != math.MaxUint32 {
		t.Fatalf("waitMillis clamp: got %d", got)
	}
}

func TestStopLeavesAProcessThatIsNotTheListedOne(t *testing.T) {
	cmd := helperCmd(t, "sleep")
	startHelper(t, cmd)
	name := trimExe(filepath.Base(os.Args[0]))
	procs, err := Processes("", name)
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(procs, func(p Process) bool { return p.PID == cmd.Process.Pid })
	if i < 0 {
		t.Fatalf("pid %d not in %+v", cmd.Process.Pid, procs)
	}
	p := procs[i]
	if p.Start.IsZero() {
		t.Fatal("listing has no start time")
	}
	p.Start = p.Start.Add(-time.Hour)
	if err := Stop(t.Context(), p, time.Second); err != nil {
		t.Fatal(err)
	}
	if !Alive(p.PID) {
		t.Fatal("Stop killed a process that was not the listed one")
	}
}

func TestTerminateWithZeroGraceDoesNotReportStillExiting(t *testing.T) {
	cmd := helperCmd(t, "sleep")
	startHelper(t, cmd)
	if err := Terminate(cmd.Process.Pid, 0); err != nil {
		t.Fatalf("Terminate with no grace: %v", err)
	}
}
