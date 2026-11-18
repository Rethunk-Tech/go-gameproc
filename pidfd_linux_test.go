//go:build linux

package gameproc

import (
	"errors"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

// A process that ignores SIGTERM must still end: the pidfd SIGKILL after grace names the same process SIGTERM did.
func TestAProcessThatIgnoresSIGTERMIsKilledWhenGraceRunsOut(t *testing.T) {
	cmd := exec.CommandContext(t.Context(), "sh", "-c", `trap "" TERM; echo ready; while :; do sleep 0.1; done`)
	r := startHelper(t, cmd)
	if line, err := r.ReadString('\n'); err != nil || line != "ready\n" {
		t.Fatalf("helper: %q, %v", line, err)
	}
	done := reap(cmd)
	if err := Terminate(cmd.Process.Pid, 200*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if got := StatusOf(<-done); got.Code != 137 || got.Signal != "SIGKILL" {
		t.Fatalf("exit %+v, want SIGKILL after grace", got)
	}
}

// Opening a pidfd of a live pid, then seeing a different start, must not signal or wait: that pid number now belongs
// to someone else. A pid that is already gone is already done.
func TestAPidfdDoesNotSignalOrWaitForAProcessThatIsNotTheListedOne(t *testing.T) {
	p := listed(t)
	wrong := p.Start.Add(-time.Hour)

	if !pidfdWait(p.PID, wrong) {
		t.Fatal("waited for a process that was not the listed one")
	}
	if !Alive(p.PID) {
		t.Fatal("the wait ended the process")
	}

	handled, err := pidfdTerminate(Process{PID: p.PID, Start: wrong}, time.Second)
	if !handled || err != nil {
		t.Fatalf("handled %v, %v; want the pidfd kept without signalling", handled, err)
	}
	if !Alive(p.PID) {
		t.Fatal("a process that was not the listed one was signalled")
	}

	if err := Terminate(p.PID, 10*time.Second); err != nil {
		t.Fatal(err)
	}
	waitGone(t, p.PID)
	if !pidfdWait(p.PID, p.Start) {
		t.Fatal("a pid that is gone was not treated as already exited")
	}
}

// Pid 0 is not a process the kernel will name with a pidfd, so Terminate's pidfd path must not claim it handled
// the signal (or the bare-pid fallback would never run on a kernel without pidfd). Init is a process we can name
// and still may not signal.
func TestAPidfdDoesNotClaimAPidTheKernelWillNotName(t *testing.T) {
	handled, err := pidfdTerminate(Process{PID: 0}, time.Millisecond)
	if handled || err != nil {
		t.Fatalf("pid 0: handled %v, %v; want a fall-through", handled, err)
	}
	if err := syscall.Kill(1, 0); !errors.Is(err, syscall.EPERM) {
		t.Skipf("pid 1 is signalable here: %v", err)
	}
	if err := Terminate(1, time.Millisecond); err == nil {
		t.Fatal("init was reported stopped")
	}
}
